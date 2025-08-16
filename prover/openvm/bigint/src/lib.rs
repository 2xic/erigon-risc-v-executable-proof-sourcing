#![no_std]

use openvm_ruint::aliases::U256;
// use openvm_bigint_guest::U256;

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
pub extern "C" fn add256_stack_scratch(
    // Pointer to first 256-bit number (8 words)
    num1_ptr: *const u32,
    // Pointer to second 256-bit number (8 words)
    num2_ptr: *const u32,
    // Pointer to store result (8 words)
    result_ptr: *mut u32,
) {
    let a = u256_from_words(num1_ptr);
    let b = u256_from_words(num2_ptr);

    let result = a + b;

    u256_to_words(&result, result_ptr);
}
