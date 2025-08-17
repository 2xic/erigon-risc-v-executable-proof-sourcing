.section .text

# Optimized 256-bit addition using a loop
.global add256_stack_scratch
add256_stack_scratch:
    li t6, 0                    # carry = 0
    li t0, 0                    # i = 0 (loop counter)
    
add_loop:
    slli t1, t0, 2              # offset = i * 4 bytes
    
    # Load num1[i]
    add t2, a0, t1              # address of num1[i]
    lw t3, 0(t2)                # load num1[i]
    
    # Load num2[i]
    add t2, a1, t1              # address of num2[i]
    lw t4, 0(t2)                # load num2[i]
    
    # Perform addition with carry
    add t2, t3, t4              # temp_sum = num1[i] + num2[i]
    add t5, t2, t6              # result = temp_sum + carry
    
    # Calculate carry for next iteration
    sltu t3, t2, t3             # carry1 = (temp_sum < num1[i])
    sltu t4, t5, t2             # carry2 = (result < temp_sum)
    or t6, t3, t4               # carry = carry1 | carry2
    
    # Store result[i]
    add t2, a2, t1              # address of result[i]
    sw t5, 0(t2)                # store result[i]
    
    # Loop control
    addi t0, t0, 1              # i++
    li t1, 8                    # number of 32-bit words in 256 bits
    blt t0, t1, add_loop        # continue if i < 8
    # Pop the first operand and result will be at top of stack.
    addi sp, sp, 32
    ret

# 256-bit NOT operation using a loop
.global not256_stack_scratch
not256_stack_scratch:
    li t0, 0                    # i = 0 (loop counter)
    
not_loop:
    slli t1, t0, 2              # offset = i * 4 bytes
    
    # Load value[i]
    add t2, a0, t1              # address of value[i]
    lw t3, 0(t2)                # load value[i]
    
    # Perform NOT operation (XOR with -1)
    li t4, -1                   # load all 1s
    xor t5, t3, t4              # result = value[i] XOR -1
    
    # Store result[i] back to same location
    sw t5, 0(t2)                # store result[i]
    
    # Loop control
    addi t0, t0, 1              # i++
    li t1, 8                    # number of 32-bit words in 256 bits
    blt t0, t1, not_loop        # continue if i < 8
    
    ret

# 256-bit shift right logical operation
# a0 = value address, a1 = shift amount address, result stored at a0
.global shr256_stack_scratch
shr256_stack_scratch:
    # Load shift amount (only use lower 32 bits)
    lw t6, 0(a1)               # shift amount
    
    # Handle shifts >= 256 (result is 0)
    li t0, 256
    bgeu t6, t0, shr_zero_result
    
    # Handle shifts >= 32 (word-level shifts)
    li t0, 32
    blt t6, t0, shr_bit_shift
    
    # Word-level shift (shift >= 32)
    div t1, t6, t0             # t1 = words to shift
    rem t6, t6, t0             # t6 = remaining bits to shift
    
    # Shift words
    li t0, 0                   # source index
    add t2, t0, t1             # destination index
    
shr_word_loop:
    li t3, 8
    bge t2, t3, shr_word_zero  # if dest >= 8, fill with zero
    
    slli t4, t2, 2             # dest offset
    add t5, a0, t4             # dest address
    
    slli t4, t0, 2             # src offset
    add t3, a0, t4             # src address
    lw t4, 0(t3)               # load src word
    sw t4, 0(t5)               # store to dest
    
    addi t0, t0, 1
    addi t2, t2, 1
    li t3, 8
    blt t0, t3, shr_word_loop
    
    j shr_bit_shift_check

shr_word_zero:
    slli t4, t0, 2             # offset
    add t5, a0, t4             # address
    sw zero, 0(t5)             # store zero
    addi t0, t0, 1
    li t3, 8
    blt t0, t3, shr_word_loop
    
shr_bit_shift_check:
    beqz t6, shr_done          # if no bit shift needed, done
    
shr_bit_shift:
    # Bit-level shift within words
    li t0, 0                   # word index
    li t1, 0                   # carry from previous word
    
shr_bit_loop:
    slli t2, t0, 2             # offset
    add t3, a0, t2             # address
    lw t4, 0(t3)               # load word
    
    # Extract bits that will be shifted out (for next word's carry)
    li t5, 32
    sub t5, t5, t6             # t5 = 32 - shift_amount
    srl t2, t4, t5             # bits to carry
    sll t2, t2, t5             # mask out unwanted bits
    
    # Perform the shift
    srl t4, t4, t6             # shift right
    or t4, t4, t1              # add carry from previous word
    sw t4, 0(t3)               # store result
    
    mv t1, t2                  # carry for next iteration
    srl t1, t1, t5             # position carry correctly
    
    addi t0, t0, 1
    li t2, 8
    blt t0, t2, shr_bit_loop
    
    j shr_done

shr_zero_result:
    # Clear all words to zero
    li t0, 0
shr_zero_loop:
    slli t1, t0, 2
    add t2, a0, t1
    sw zero, 0(t2)
    addi t0, t0, 1
    li t1, 8
    blt t0, t1, shr_zero_loop

shr_done:
    # Pop the shift amount, leave result on stack
    addi sp, sp, 32
    ret

# 256-bit signed less than comparison
# a0 = first value address, a1 = second value address
# Result (0 or 1) stored at a1 address
.global slt256_stack_scratch
slt256_stack_scratch:
    # Load most significant words to check sign
    lw t0, 28(a0)              # MSW of first value
    lw t1, 28(a1)              # MSW of second value
    
    # Extract sign bits
    srai t2, t0, 31            # sign of first (-1 if negative, 0 if positive)
    srai t3, t1, 31            # sign of second
    
    # If signs differ, result is determined by signs
    bne t2, t3, slt_sign_diff
    
    # Same signs - compare magnitude from MSW to LSW
    li t4, 7                   # start from word 7 (MSW)
    
slt_compare_loop:
    slli t5, t4, 2             # offset
    add t0, a0, t5             # address of word in first value
    add t1, a1, t5             # address of word in second value
    lw t2, 0(t0)               # load word from first
    lw t3, 0(t1)               # load word from second
    
    bltu t2, t3, slt_true      # if first < second (unsigned), result is true
    bgtu t2, t3, slt_false     # if first > second (unsigned), result is false
    
    # Words are equal, continue to next word
    addi t4, t4, -1
    bgez t4, slt_compare_loop
    
    # All words equal
    j slt_false

slt_sign_diff:
    # First negative, second positive -> true
    # First positive, second negative -> false
    bltz t2, slt_true          # if first is negative, result is true
    j slt_false

slt_true:
    li t0, 1
    j slt_store_result

slt_false:
    li t0, 0

slt_store_result:
    # Clear second value and store result
    sw t0, 0(a1)
    sw zero, 4(a1)
    sw zero, 8(a1)
    sw zero, 12(a1)
    sw zero, 16(a1)
    sw zero, 20(a1)
    sw zero, 24(a1)
    sw zero, 28(a1)
    
    # Pop first operand
    addi sp, sp, 32
    ret

# 256-bit equality comparison
# a0 = first value address, a1 = second value address
# Result (0 or 1) stored at a1 address
.global eq256_stack_scratch
eq256_stack_scratch:
    li t0, 0                   # word index
    
eq_compare_loop:
    slli t1, t0, 2             # offset
    add t2, a0, t1             # address in first value
    add t3, a1, t1             # address in second value
    lw t4, 0(t2)               # load from first
    lw t5, 0(t3)               # load from second
    
    bne t4, t5, eq_false       # if any word differs, not equal
    
    addi t0, t0, 1
    li t1, 8
    blt t0, t1, eq_compare_loop
    
    # All words equal
    li t0, 1
    j eq_store_result

eq_false:
    li t0, 0

eq_store_result:
    # Clear second value and store result
    sw t0, 0(a1)
    sw zero, 4(a1)
    sw zero, 8(a1)
    sw zero, 12(a1)
    sw zero, 16(a1)
    sw zero, 20(a1)
    sw zero, 24(a1)
    sw zero, 28(a1)
    
    # Pop first operand
    addi sp, sp, 32
    ret

# 256-bit unsigned greater than comparison
# a0 = first value address, a1 = second value address
# Result (0 or 1) stored at a1 address
.global gt256_stack_scratch
gt256_stack_scratch:
    # Compare from MSW to LSW
    li t4, 7                   # start from word 7 (MSW)
    
gt_compare_loop:
    slli t5, t4, 2             # offset
    add t0, a0, t5             # address of word in first value
    add t1, a1, t5             # address of word in second value
    lw t2, 0(t0)               # load word from first
    lw t3, 0(t1)               # load word from second
    
    bgtu t2, t3, gt_true       # if first > second (unsigned), result is true
    bltu t2, t3, gt_false      # if first < second (unsigned), result is false
    
    # Words are equal, continue to next word
    addi t4, t4, -1
    bgez t4, gt_compare_loop
    
    # All words equal
    j gt_false

gt_true:
    li t0, 1
    j gt_store_result

gt_false:
    li t0, 0

gt_store_result:
    # Clear second value and store result
    sw t0, 0(a1)
    sw zero, 4(a1)
    sw zero, 8(a1)
    sw zero, 12(a1)
    sw zero, 16(a1)
    sw zero, 20(a1)
    sw zero, 24(a1)
    sw zero, 28(a1)
    
    # Pop first operand
    addi sp, sp, 32
    ret

# 256-bit unsigned less than comparison
# a0 = first value address, a1 = second value address
# Result (0 or 1) stored at a1 address
.global lt256_stack_scratch
lt256_stack_scratch:
    # Compare from MSW to LSW
    li t4, 7                   # start from word 7 (MSW)
    
lt_compare_loop:
    slli t5, t4, 2             # offset
    add t0, a0, t5             # address of word in first value
    add t1, a1, t5             # address of word in second value
    lw t2, 0(t0)               # load word from first
    lw t3, 0(t1)               # load word from second
    
    bltu t2, t3, lt_true       # if first < second (unsigned), result is true
    bgtu t2, t3, lt_false      # if first > second (unsigned), result is false
    
    # Words are equal, continue to next word
    addi t4, t4, -1
    bgez t4, lt_compare_loop
    
    # All words equal
    j lt_false

lt_true:
    li t0, 1
    j lt_store_result

lt_false:
    li t0, 0

lt_store_result:
    # Clear second value and store result
    sw t0, 0(a1)
    sw zero, 4(a1)
    sw zero, 8(a1)
    sw zero, 12(a1)
    sw zero, 16(a1)
    sw zero, 20(a1)
    sw zero, 24(a1)
    sw zero, 28(a1)
    
    # Pop first operand
    addi sp, sp, 32
    ret

# 256-bit memory store operation
# a0 = address (from top of stack), a1 = value address (second on stack)
.global mstore256_stack_scratch
mstore256_stack_scratch:
    # Load the memory address (only use lower 32 bits for now)
    lw t0, 0(a0)               # memory address
    
    # Store all 8 32-bit words from value to memory
    li t1, 0                   # word index
    
mstore_loop:
    slli t2, t1, 2             # offset = word_index * 4
    add t3, a1, t2             # address of value[word_index]
    add t4, t0, t2             # address of memory[addr + word_index]
    lw t5, 0(t3)               # load value word
    sw t5, 0(t4)               # store to memory
    
    addi t1, t1, 1             # word_index++
    li t2, 8                   # 8 words in 256 bits
    blt t1, t2, mstore_loop    # continue if word_index < 8
    
    # Pop both operands (address and value)
    addi sp, sp, 64
    ret

# 256-bit memory load operation  
# a0 = address (from top of stack), result stored at address location on stack
.global mload256_stack_scratch
mload256_stack_scratch:
    # Load the memory address (only use lower 32 bits for now)
    lw t0, 0(a0)               # memory address
    
    # Load all 8 32-bit words from memory to stack location
    li t1, 0                   # word index
    
mload_loop:
    slli t2, t1, 2             # offset = word_index * 4
    add t3, t0, t2             # address of memory[addr + word_index]
    add t4, a0, t2             # address of stack[word_index] (reuse address location)
    lw t5, 0(t3)               # load from memory
    sw t5, 0(t4)               # store to stack
    
    addi t1, t1, 1             # word_index++
    li t2, 8                   # 8 words in 256 bits
    blt t1, t2, mload_loop     # continue if word_index < 8
    
    ret