#![no_std]

use openvm_ruint::aliases::U256;

// Utility functions for working with U256 from assembly
#[unsafe(no_mangle)]
extern "C" fn u256_from_words(words: *const u32) -> U256 {
    unsafe {
        let word_slice = core::slice::from_raw_parts(words, 8);
        let mut bytes = [0u8; 32];

        // Convert 8 u32 words to 32 bytes (little-endian)
        for (i, &word) in word_slice.iter().enumerate() {
            let word_bytes = word.to_le_bytes();
            bytes[i * 4..(i + 1) * 4].copy_from_slice(&word_bytes);
        }

        U256::from_le_bytes(bytes)
    }
}

#[unsafe(no_mangle)]
extern "C" fn u256_to_words(value: *const U256, words: *mut u32) {
    unsafe {
        let bytes: [u8; 32] = (*value).to_le_bytes();
        let word_slice = core::slice::from_raw_parts_mut(words, 8);

        // Convert 32 bytes to 8 u32 words (little-endian)
        for i in 0..8 {
            word_slice[i] = u32::from_le_bytes([
                bytes[i * 4],
                bytes[i * 4 + 1],
                bytes[i * 4 + 2],
                bytes[i * 4 + 3],
            ]);
        }
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn openvm_add256_stack_scratch(num1_ptr: *const u32, num2_ptr: *mut u32) {
    let a = u256_from_words(num1_ptr);
    let b = u256_from_words(num2_ptr);

    let result = a + b;

    u256_to_words(&result, num2_ptr);
}

#[unsafe(no_mangle)]
pub extern "C" fn openvm_eq256_stack_scratch(num1_ptr: *const u32, num2_ptr: *mut u32) {
    let a = u256_from_words(num1_ptr);
    let b = u256_from_words(num2_ptr);
    let result = if a == b {
        U256::from(1u32)
    } else {
        U256::from(0u32)
    };
    u256_to_words(&result, num2_ptr);
}

#[unsafe(no_mangle)]
pub extern "C" fn openvm_lt256_stack_scratch(num1_ptr: *const u32, num2_ptr: *mut u32) {
    let a = u256_from_words(num1_ptr);
    let b = u256_from_words(num2_ptr);
    let result = if a < b {
        U256::from(1u32)
    } else {
        U256::from(0u32)
    };
    u256_to_words(&result, num2_ptr);
}

#[unsafe(no_mangle)]
pub extern "C" fn openvm_gt256_stack_scratch(num1_ptr: *const u32, num2_ptr: *mut u32) {
    let a = u256_from_words(num1_ptr);
    let b = u256_from_words(num2_ptr);
    let result = if a > b {
        U256::from(1u32)
    } else {
        U256::from(0u32)
    };
    u256_to_words(&result, num2_ptr);
}

#[unsafe(no_mangle)]
pub extern "C" fn openvm_shr256_stack_scratch(value_ptr: *const u32, shift_ptr: *mut u32) {
    let value = u256_from_words(value_ptr);
    let shift = u256_from_words(shift_ptr);
    let result = value >> shift;
    u256_to_words(&result, shift_ptr);
}

#[unsafe(no_mangle)]
pub extern "C" fn openvm_not256_stack_scratch(value_ptr: *mut u32) {
    let value = u256_from_words(value_ptr);
    let result = !value;
    u256_to_words(&result, value_ptr);
}
