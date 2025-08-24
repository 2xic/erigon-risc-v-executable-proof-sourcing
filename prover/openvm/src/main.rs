#[allow(unused_imports, clippy::single_component_path_imports)]
use bigint;
use openvm::io::{read, reveal_u32};
use std::arch::global_asm;

// Simple memory simulation using a static array (limited but functional)
// static mut MEMORY: [[u32; 8]; 1024] = [[0u32; 8]; 1024];

global_asm!(include_str!("./risc.asm"));

// Create wrapper functions that are easier to call from assembly
#[unsafe(no_mangle)]
extern "C" fn read_u64_func() -> u64 {
    read::<u64>()
}

#[unsafe(no_mangle)]
extern "C" fn reveal_u32_func(val: u32, idx: u32) {
    reveal_u32(val, idx as usize);
}

/*
#[unsafe(no_mangle)]
pub extern "C" fn mstore256_stack_scratch(addr_ptr: *const u32, value_ptr: *const u32) {
    unsafe {
        // Only use the lowest 32-bit word as address, ignore higher-order bits
        let raw_addr = *addr_ptr;

        // Mask to reasonable range and convert to word index
        let addr = ((raw_addr as usize) & 0x7FFFF) / 32; // Limit to ~4MB range, convert to word index

        if addr < MEMORY.len() {
            // Copy 8 u32 words from value_ptr to memory
            for i in 0..8 {
                MEMORY[addr][i] = *(value_ptr.add(i));
            }
        }
        // Silently ignore out-of-bounds writes
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn mload256_stack_scratch(addr_ptr: *const u32, result_ptr: *mut u32) {
    unsafe {
        // Only use the lowest 32-bit word as address, ignore higher-order bits
        let raw_addr = *addr_ptr;

        // Mask to reasonable range and convert to word index
        let addr = ((raw_addr as usize) & 0x7FFFF) / 32; // Limit to ~4MB range, convert to word index

        let value = if addr < MEMORY.len() {
            MEMORY[addr]
        } else {
            [0u32; 8] // Return zeros for out-of-bounds access
        };

        // Copy 8 u32 words to result_ptr
        for i in 0..8 {
            *(result_ptr.add(i)) = value[i];
        }
    }
}
*/

unsafe extern "C" {
    fn execute();
}

fn main() {
    unsafe {
        execute();
    }
}
