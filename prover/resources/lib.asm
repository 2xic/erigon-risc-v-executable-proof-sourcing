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
    
    ret

# Convert 8 u32 words to 32 bytes (little-endian)
# a0=words_ptr, a1=bytes_ptr
.global u256_from_words
u256_from_words:
    li t0, 0                    # i = 0
    
from_words_loop:
    slli t1, t0, 2              # offset = i * 4
    add t2, a0, t1
    lw t3, 0(t2)                # word = words[i]
    
    add t2, a1, t1              # bytes + offset
    sb t3, 0(t2)                # bytes[0] = word & 0xFF
    srli t4, t3, 8
    sb t4, 1(t2)                # bytes[1] = (word >> 8) & 0xFF
    srli t4, t3, 16  
    sb t4, 2(t2)                # bytes[2] = (word >> 16) & 0xFF
    srli t4, t3, 24
    sb t4, 3(t2)                # bytes[3] = (word >> 24) & 0xFF
    
    addi t0, t0, 1
    li t1, 8
    blt t0, t1, from_words_loop
    ret

# Convert 32 bytes to 8 u32 words (little-endian)
# a0=bytes_ptr, a1=words_ptr  
.global u256_to_words
u256_to_words:
    li t0, 0                    # i = 0
    
to_words_loop:
    slli t1, t0, 2              # offset = i * 4
    add t2, a0, t1              # bytes + offset
    
    lbu t3, 0(t2)               # b0 = bytes[0]
    lbu t4, 1(t2)               # b1 = bytes[1] 
    slli t4, t4, 8
    or t3, t3, t4               # word = b0 | (b1 << 8)
    
    lbu t4, 2(t2)               # b2 = bytes[2]
    slli t4, t4, 16
    or t3, t3, t4               # word |= (b2 << 16)
    
    lbu t4, 3(t2)               # b3 = bytes[3]
    slli t4, t4, 24  
    or t3, t3, t4               # word |= (b3 << 24)
    
    add t2, a1, t1              # words + offset
    sw t3, 0(t2)                # words[i] = word
    
    addi t0, t0, 1
    li t1, 8
    blt t0, t1, to_words_loop
    ret