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
