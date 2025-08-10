use openvm::io::{read, reveal_u32};
use std::arch::global_asm;

mod bigint;

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

unsafe extern "C" {
    fn execute();
}

fn main() {
    unsafe {
        execute();
    }
}
