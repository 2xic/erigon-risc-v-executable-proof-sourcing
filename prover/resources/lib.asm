.section .text

# Add two 256-bit numbers (8 u32 words each)
# a0=num1_ptr, a1=num2_ptr, a2=result_ptr
.global add256_stack_scratch
add256_stack_scratch:
    addi sp, sp, -12
    sw s0, 0(sp)
    sw s1, 4(sp) 
    sw ra, 8(sp)
    
    li s1, 0                    # carry = 0
    li t0, 0                    # i = 0
    
add_loop:
    slli t1, t0, 2              # offset = i * 4
    add t2, a0, t1
    lw t3, 0(t2)                # num1[i]
    add t2, a1, t1  
    lw t4, 0(t2)                # num2[i]
    
    add t5, t3, t4              # sum = num1[i] + num2[i]
    add t6, t5, s1              # result = sum + carry
    
    sltu s0, t5, t3             # carry1 = sum < num1[i]
    sltu s1, t6, t5             # carry2 = result < sum  
    or s1, s0, s1               # carry = carry1 | carry2
    
    add t2, a2, t1
    sw t6, 0(t2)                # store result[i]
    
    addi t0, t0, 1
    li t1, 8
    blt t0, t1, add_loop
    
    lw s0, 0(sp)
    lw s1, 4(sp)
    lw ra, 8(sp)
    addi sp, sp, 12
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