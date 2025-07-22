package main

import (
	"encoding/binary"
	"fmt"
	"testing"

	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon/core/vm"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	uc "github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

const EbreakInstr = 0x00100073

type TestRunner struct {
	program []byte
}

func NewTestRunner(program []byte) *TestRunner {
	return &TestRunner{
		program: program,
	}
}

func (t *TestRunner) Execute() (*AssemblyFile, error) {
	contractAddr := libcommon.HexToAddress("0x1234567890123456789012345678901234567890")

	runner := NewSimpleTracer()
	err := runner.DeployContract(contractAddr, t.program, uint256.NewInt(1000))
	if err != nil {
		return nil, err
	}

	transpiler, _, err := runner.ExecuteContract(contractAddr, nil, 100000)
	if err != nil {
		return nil, err
	}

	assembly := transpiler.toAssembly()
	return assembly, nil
}

type VmRunner struct{}

type ExecutionResult struct {
	stackSnapshots *[][]uint64
}

func NewVmRunner() (*VmRunner, error) {
	return &VmRunner{}, nil
}

func (vm *VmRunner) Execute(bytecode []byte) (*ExecutionResult, error) {
	mu, err := uc.NewUnicorn(uc.ARCH_RISCV, uc.MODE_RISCV64)
	if err != nil {
		return nil, err
	}

	codeAddr := uint64(0x10000)
	stackAddr := uint64(0x7fff0000)
	memSize := uint64(0x10000)

	err = mu.MemMap(codeAddr, memSize)
	if err != nil {
		return nil, err
	}
	err = mu.MemMap(stackAddr, memSize)
	if err != nil {
		return nil, err
	}

	stackTop := stackAddr + memSize - 16
	err = mu.RegWrite(uc.RISCV_REG_SP, stackTop)
	if err != nil {
		return nil, err
	}

	allStackSnapshots := make([][]uint64, 0)
	executionResults := &ExecutionResult{
		stackSnapshots: &allStackSnapshots,
	}

	hook, err := mu.HookAdd(uc.HOOK_CODE, func(mu uc.Unicorn, addr uint64, size uint32) {
		// Read the instruction at this address
		instrBytes, err := mu.MemRead(addr, 4)
		if err != nil {
			panic(err)
		}
		instr := binary.LittleEndian.Uint32(instrBytes)

		if instr == uint32(EbreakInstr) {
			snapshot, err := printStackState(mu, stackAddr, memSize)
			if err != nil {
				panic(err)
			}

			allStackSnapshots = append(allStackSnapshots, snapshot)
			pc, err := mu.RegRead(uc.RISCV_REG_PC)
			if err != nil {
				panic(err)
			}
			// Skip 4 bytes (EBREAK instruction size)
			err = mu.RegWrite(uc.RISCV_REG_PC, pc+4)
			if err != nil {
				panic(err)
			}
		}
	}, 1, 0)

	if err != nil {
		return nil, err
	}

	err = mu.MemWrite(codeAddr, bytecode)
	if err != nil {
		return nil, err
	}
	err = mu.Start(codeAddr, codeAddr+uint64(len(bytecode)))
	if err != nil {
		return nil, err
	}

	err = mu.HookDel(hook)
	if err != nil {
		return nil, err
	}
	err = mu.Close()

	if err != nil {
		return nil, err
	}
	return executionResults, nil
}

func printStackState(mu uc.Unicorn, stackBase, memSize uint64) ([]uint64, error) {
	sp, _ := mu.RegRead(uc.RISCV_REG_SP)
	stackTop := stackBase + memSize - 16

	if sp > stackTop {
		return nil, nil
	}
	numEntries := (stackTop - sp) / 8
	stack := make([]uint64, numEntries)
	for i := uint64(0); i < numEntries; i++ {
		addr := sp + (i * 8)
		data, err := mu.MemRead(addr, 8)
		if err != nil {
			return nil, err
		}
		value := binary.LittleEndian.Uint64(data)
		stack[i] = value
	}
	return stack, nil
}

func TestAddOpcode(t *testing.T) {
	bytecode := []byte{
		byte(vm.PUSH1), 0x42,
		byte(vm.PUSH1), 0x01,
		byte(vm.ADD),
	}
	assembly, err := NewTestRunner(bytecode).Execute()
	assert.NoError(t, err)
	bytecode, err = assembly.assembleToBytecode()
	assert.NoError(t, err)

	execution, err := NewVmRunner()
	assert.NoError(t, err)
	snapshot, err := execution.Execute(bytecode)
	assert.NoError(t, err)

	fmt.Println(snapshot.stackSnapshots)
	snapShot := *snapshot.stackSnapshots
	assert.Len(t, snapShot, 3)
	assert.Equal(t, []uint64{0x42}, snapShot[0])
	assert.Equal(t, []uint64{0x01, 0x42}, snapShot[1])
	assert.Equal(t, []uint64{0x43}, snapShot[2])

}

func TestZkProverConnection(t *testing.T) {
	// From https://gist.github.com/2xic/82ff5065eff396f063c60bb4a281034b
	content := `
.global execute
execute:
	# Save registers and setup stack
	addi sp, sp, -32
	sw ra, 28(sp)
	sw s0, 24(sp)
	sw s1, 20(sp)

	# Call read() to get n
	call read_u64_func
	mv s0, a0 # s0 = n (save in callee-saved register)

	# Initialize fibonacci variables
	li t1, 0  # a = 0
	li t2, 1  # b = 1
	mv t0, s0 # counter = n

	# Check if n == 0
	beqz t0, 3f

	# Fibonacci loop
2:
	add t3, t1, t2  # c = a + b
	mv t1, t2       # a = b
	mv t2, t3       # b = c
	addi t0, t0, -1 # counter--
	bnez t0, 2b     # loop if counter != 0

	# Call reveal_u32(a as u32, 0) - lower 32 bits
3:
	mv s1, t1 # save result in s1
	mv a0, s1 # a0 = result
	li a1, 0  # index = 0
	call reveal_u32_func

	# Call reveal_u32((a >> 32) as u32, 1) - upper 32 bits
	li t4, 32
	srl a0, s1, t4 # a0 = result >> 32
	li a1, 1       # index = 1
	call reveal_u32_func

	# Restore registers and return
	lw s1, 20(sp)
	lw s0, 24(sp)
	lw ra, 28(sp)
	addi sp, sp, 32
	ret	
	`
	zkVm := NewZkProver(content)
	output, err := zkVm.Prove()
	assert.NoError(t, err)
	assert.Equal(t, "Execution output: [55, 0, 0, 0, 55, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]\n", output)
}
