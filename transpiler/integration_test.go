package main

import (
	"encoding/binary"
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
	bytecode, err = assembly.toBytecode()
	assert.NoError(t, err)

	execution, err := NewVmRunner()
	assert.NoError(t, err)
	snapshot, err := execution.Execute(bytecode)
	assert.NoError(t, err)

	// Verify that the stack is as expected at each step of the execution
	snapShot := *snapshot.stackSnapshots
	assert.Len(t, snapShot, 3)
	assert.Equal(t, []uint64{0x42}, snapShot[0])
	assert.Equal(t, []uint64{0x01, 0x42}, snapShot[1])
	assert.Equal(t, []uint64{0x43}, snapShot[2])

	// Verify that we can run the Zk prover on the assembly
	content, err := assembly.toToolChainCompatibleAssembly()
	assert.NoError(t, err)
	zkVm := NewZkProver(content)
	output, err := zkVm.TestRun()
	assert.NoError(t, err)
	// All zero as we don't write any of the output.
	assert.Equal(t, "Execution output: [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]\n", output)

	// output, err = zkVm.Prove()
	// assert.NoError(t, err)
	//	assert.Contains(t, "app_pk commit: 0x0094295cb5d90deb2b28cab4d658dab0fdc2922c4e9c10305bbf277c8d29d881\n", output)
	//	assert.Contains(t, "exe commit: 0x0086d334e8f5715dd186700497c4b3d3c667cd812fda3135c6414c66eb0fc0e3\n", output)
}
