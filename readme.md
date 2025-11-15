# Erigon: RISCV Executable Proof Sourcing

Project repository for implementing the [Erigon: RISCV Executable Proof Sourcing](https://github.com/2xic/cohort-six/blob/add-project-proposal/projects/erigon_riscv_proof_sourcing.md) project. This was implemented part of EPF cohort six and you can read the final report [here](https://hackmd.io/@2xic/B1_wo07lZx).

## Resources

### Documents
Some documents that might be of interests:
- [Setup and install](./docs/setup_and_install.md)
- [CLI Tool](./docs/cli.md)
- [Testing setup](./docs/testing_setup.md)
- [Transpiler status](./docs/transpiler_status.md)

### Project specific
- [Development updates](https://github.com/eth-protocol-fellows/cohort-six/blob/master/development-updates.md) 
- [Basic Erigon opcode tracer](https://gist.github.com/2xic/1bcccc8cf74419ae0c837fce03285625) for experimenting with Erigon integration.
- [Basic OpenVM toolchain](https://gist.github.com/2xic/82ff5065eff396f063c60bb4a281034b) for proving RISC-V assembly directly.

### Dependencies
- [Erigon](https://github.com/erigontech/erigon) for the EVM execution.
- [OpenVm](https://blog.openvm.dev/) for the proving of the RISC-V executable.

### Testing Dependencies
- [Unicorn](https://www.unicorn-engine.org/) for RISC-V emulation.

### Proving using stateless execution
- [SP1 Hypercube](https://blog.succinct.xyz/sp1-hypercube/)
- [Ress: Scaling Ethereum with Stateless Reth Nodes](https://www.paradigm.xyz/2025/03/stateless-reth-nodes)
- I also wrote a small section on it on [my blog](https://2xic.xyz/blog/zkvm.html#debug_executionWitness)

### ZkVm
- [vnTinyRAM](https://blog.plan99.net/vntinyram-7b9d5b299097) - understanding zkVMs proofs from the ground up (after having read [Quadratic Arithmetic Programs: from Zero to Hero](https://medium.com/@VitalikButerin/quadratic-arithmetic-programs-from-zero-to-hero-f6d558cea649#.ghchc7urv)).
- [Ground Up Guide: zkEVM, EVM Compatibility & Rollups](https://www.immutable.com/blog/ground-up-guide-zkevm-evm-compatibility-rollups) - a bit outdate, but had some good context on how some ZkEVMs (used to) works. 
- [Long-term L1 execution layer proposal: replace the EVM with RISC-V ](https://ethereum-magicians.org/t/long-term-l1-execution-layer-proposal-replace-the-evm-with-risc-v/23617)
- [What is the best ISA for Ethereum?](https://hackmd.io/@leoalt/best-isa-ethereum)
- [RISC-V ZKVMs: the Good and the Bad](https://argument.xyz/blog/riscv-good-bad/)
- [The future of ZK is in RISC-V zkVMs, but the industry must be careful: how Succinct's SP1's departure from standards causes bugs](https://blog.lambdaclass.com/the-future-of-zk-is-in-risc-v-zkvms-but-the-industry-must-be-careful-how-succincts-sp1s-departure-from-standards-causes-bugs/)
- [Ethproofs Call #4 | enshrine RISC-V?](https://www.youtube.com/watch?v=rJiEV7jJFl4)

### Native rollups
Some of the articles on Native rollups were useful as a way to think about proof assumptions. They aren't fully related to the project, but as I found them useful when working on this project I'm sharing.
- [Native rollups—superpowers from L1 execution](https://ethresear.ch/t/native-rollups-superpowers-from-l1-execution/21517)
- [Revisit Native Rollups](https://paragraph.com/@taiko-labs/revisit-native-rollups)
- [Revisit Native Rollups](https://paragraph.com/@taiko-labs/revisit-native-rollups)


