package main

import (
	"erigon-transpiler-risc-v/prover"
	"testing"

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
	addi a0, sp, 0        # First number
	addi a1, sp, 32       # Second number  
	addi a2, sp, 0        # Result (overwrite first)
	call add256_stack_scratch
	
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
