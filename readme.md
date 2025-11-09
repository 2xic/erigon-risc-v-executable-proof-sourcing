# Erigon: RISCV Executable Proof Sourcing

Project repository for implementing the [Erigon: RISCV Executable Proof Sourcing](https://github.com/2xic/cohort-six/blob/add-project-proposal/projects/erigon_riscv_proof_sourcing.md) project.

Current status will be updated in the [Opcode status](./docs/opcode_status.md) document.

## Setup
```bash
git submodule update --init --recursive
```

## Resources

### Documents
Some documents that might be of interests:
- [CLI Tool](./docs/cli.md)
- [Testing setup](./docs/testing_setup.md)
- [Opcode status](./docs/opcode_status.md)

### Project specific
- [Development updates](https://github.com/eth-protocol-fellows/cohort-six/blob/master/development-updates.md) 
- [Basic Erigon opcode tracer](https://gist.github.com/2xic/1bcccc8cf74419ae0c837fce03285625) for experimenting with Erigon integration.
- [Basic OpenVM toolchain](https://gist.github.com/2xic/82ff5065eff396f063c60bb4a281034b) for proving RISC-V assembly directly.

### Dependencies
- [Erigon](https://github.com/erigontech/erigon) for the EVM execution.
- [OpenVm](https://blog.openvm.dev/) for the proving of the RISC-V executable.

### Testing Dependencies
- [Unicorn](https://www.unicorn-engine.org/) for RISC-V emulation.

### ZkVm
- [vnTinyRAM](https://blog.plan99.net/vntinyram-7b9d5b299097) - understanding zkVMs proofs from the ground up (after having read [Quadratic Arithmetic Programs: from Zero to Hero](https://medium.com/@VitalikButerin/quadratic-arithmetic-programs-from-zero-to-hero-f6d558cea649#.ghchc7urv)).
- [Ground Up Guide: zkEVM, EVM Compatibility & Rollups](https://www.immutable.com/blog/ground-up-guide-zkevm-evm-compatibility-rollups) - a bit outdate, but had some good context on how some ZkEVMs (used to) works. 
- [Long-term L1 execution layer proposal: replace the EVM with RISC-V ](https://ethereum-magicians.org/t/long-term-l1-execution-layer-proposal-replace-the-evm-with-risc-v/23617)
- [What is the best ISA for Ethereum?](https://hackmd.io/@leoalt/best-isa-ethereum)
- [RISC-V ZKVMs: the Good and the Bad](https://argument.xyz/blog/riscv-good-bad/)
