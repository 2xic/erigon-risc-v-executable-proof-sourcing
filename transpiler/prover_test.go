package main

import (
	"erigon-transpiler-risc-v/prover"
	"fmt"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func Test256BitStack(t *testing.T) {
	/*
		# The same logic as the code below, but in python
		a = 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF
		b = 0x1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF
		result_256bit = (a + b) % (2**256)
		print(hex(result_256bit))

		# The output from the code below.
		output = [238, 205, 171, 144, 120, 86, 52, 18, 239, 205, 171, 144, 120, 86, 52, 18, 239, 205, 171, 144, 120, 86, 52, 18, 239, 205, 171, 144, 120, 86, 52, 18]
		result_bytes = output[:32]
		result_bytes_be = result_bytes[::-1]
		hex_be = bytes(result_bytes_be).hex()
		print(f"Big-endian hex: 0x{hex_be}")
	*/
	content := `
.text
execute:
	addi sp, sp, -80      # Allocate space for two 256-bit numbers
	sw ra, 76(sp)         # Save return address
	
	# Create first 256-bit number: 0xFFFF...FFFF (all bits set)
	li t0, 0xFFFFFFFF
	sw t0, 0(sp)          # Store in all 8 words
	sw t0, 4(sp)
	sw t0, 8(sp)
	sw t0, 12(sp)
	sw t0, 16(sp)
	sw t0, 20(sp)
	sw t0, 24(sp)
	sw t0, 28(sp)
	
	# Create second 256-bit number: repeating pattern
	li t0, 0x90ABCDEF     # Low word of pattern
	li t1, 0x12345678     # High word of pattern
	sw t0, 32(sp)         # Alternate the pattern
	sw t1, 36(sp)
	sw t0, 40(sp)
	sw t1, 44(sp)
	sw t0, 48(sp)
	sw t1, 52(sp)
	sw t0, 56(sp)
	sw t1, 60(sp)
	
	# Perform 256-bit addition
	addi a0, sp, 32        # First number
	addi a1, sp, 0         # Second number  
	call openvm_add256_stack_scratch
	
	# Reveal results
	lw a0, 0(sp)
	li a1, 0
	call reveal_u32_func
	
	lw a0, 4(sp)
	li a1, 1
	call reveal_u32_func
	
	lw a0, 8(sp)
	li a1, 2
	call reveal_u32_func
	
	lw a0, 12(sp)
	li a1, 3
	call reveal_u32_func
	
	lw a0, 16(sp)
	li a1, 4
	call reveal_u32_func
	
	lw a0, 20(sp)
	li a1, 5
	call reveal_u32_func
	
	lw a0, 24(sp)
	li a1, 6
	call reveal_u32_func
	
	lw a0, 28(sp)
	li a1, 7
	call reveal_u32_func
	
	lw ra, 76(sp)         # Restore return address
	addi sp, sp, 80       # Restore stack
	ret
	`
	zkVm := prover.NewZkProver(content)
	output, err := zkVm.TestRun()
	assert.NoError(t, err)
	assert.Equal(t, "Execution output: [238, 205, 171, 144, 120, 86, 52, 18, 239, 205, 171, 144, 120, 86, 52, 18, 239, 205, 171, 144, 120, 86, 52, 18, 239, 205, 171, 144, 120, 86, 52, 18]", output)
}

func TestDataSectionConstant(t *testing.T) {
	// Test storing a 256-bit constant in data section and reading it back
	content := `
.section .data
test_constant:
    .word 0x12345678
    .word 0x9ABCDEF0
    .word 0x11111111
    .word 0x22222222
    .word 0x33333333
    .word 0x44444444
    .word 0x55555555
    .word 0x66666666

.section .text
.global execute
execute:
	# Save stack and return address
	mv s2, sp
	mv s1, ra
	
	# Allocate 32 bytes on stack for 256-bit value
	addi sp, sp, -32
	
	# Load address of test constant
	la t0, test_constant
	
	# Load all 8 32-bit words from data section to stack
	lw t1, 0(t0)
	sw t1, 0(sp)
	lw t1, 4(t0)
	sw t1, 4(sp)
	lw t1, 8(t0)
	sw t1, 8(sp)
	lw t1, 12(t0)
	sw t1, 12(sp)
	lw t1, 16(t0)
	sw t1, 16(sp)
	lw t1, 20(t0)
	sw t1, 20(sp)
	lw t1, 24(t0)
	sw t1, 24(sp)
	lw t1, 28(t0)
	sw t1, 28(sp)
	
	# Reveal the 8 32-bit words
	lw a0, 0(sp)
	li a1, 0
	call reveal_u32_func
	
	lw a0, 4(sp)
	li a1, 1
	call reveal_u32_func
	
	lw a0, 8(sp)
	li a1, 2
	call reveal_u32_func
	
	lw a0, 12(sp)
	li a1, 3
	call reveal_u32_func
	
	lw a0, 16(sp)
	li a1, 4
	call reveal_u32_func
	
	lw a0, 20(sp)
	li a1, 5
	call reveal_u32_func
	
	lw a0, 24(sp)
	li a1, 6
	call reveal_u32_func
	
	lw a0, 28(sp)
	li a1, 7
	call reveal_u32_func
	
	# Restore stack and return address
	mv sp, s2
	mv ra, s1
	ret
	`
	zkVm := prover.NewZkProver(content)
	output, err := zkVm.TestRun()
	assert.NoError(t, err)
	// Expected: 0x12345678, 0x9ABCDEF0, 0x11111111, 0x22222222, 0x33333333, 0x44444444, 0x55555555, 0x66666666
	assert.Equal(t, "Execution output: [120, 86, 52, 18, 240, 222, 188, 154, 17, 17, 17, 17, 34, 34, 34, 34, 51, 51, 51, 51, 68, 68, 68, 68, 85, 85, 85, 85, 102, 102, 102, 102]", output)
}

func TestSolidityCompilation(t *testing.T) {
	counterSource := `
		pragma solidity ^0.8.26;

		contract Counter {
			uint256 public count;

			function get() public view returns (uint256) {
				return count;
			}

			function inc() public {
				count += 1;
			}

			function dec() public {
				count -= 1;
			}
		}
	`

	bytecode, err := prover.CompileSolidity(counterSource, "Counter")
	if err != nil {
		t.Fatalf("Failed to compile Solidity: %v", err)
	}

	if len(bytecode) == 0 {
		t.Fatal("Expected non-empty bytecode")
	}

	callData := prover.EncodeCallData("inc")
	assembly, _, err := NewTestRunnerWithConfig(bytecode, TestConfig{
		CallValue: uint256.NewInt(0),
		CallData:  callData,
	}).Execute()
	assert.NoError(t, err)
	fmt.Println(assembly)

	content, err := assembly.ToToolChainCompatibleAssembly()
	assert.NoError(t, err)
	fmt.Println(content)

	zkVm := prover.NewZkProver(content)
	output, err := zkVm.TestRun()
	assert.NoError(t, err)
	// All zero as we don't write any of the output.
	assert.Equal(t, "Execution output: [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]", output)
}
