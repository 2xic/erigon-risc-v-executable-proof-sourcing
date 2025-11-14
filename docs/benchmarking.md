# Benchmarking

We benchmark the Erigon block-prover against three zkVM-based alternatives. You can read the results in the final report [here](https://hackmd.io/@2xic/B1_wo07lZx).

## Setups Tested

- **Erigon block-prover**: Native implementation with STARK proof support
- **[Ethrex](https://github.com/lambdaclass/ethrex)**: EthRex stateless Proving using SP1 zkVM prover - [benchmarking/ethrex-prover-bench/](../benchmarking/ethrex-prover-bench/)
- **[RSP](https://github.com/succinctlabs/rsp)**: Reth Stateless Proving using SP1 zkVM prover - [benchmarking/rsp-prover-bench/](../benchmarking/rsp-prover-bench/)
- **[Zeth](https://github.com/risc0/zeth)**: Reth Stateless Proving using RISC0 zkVM prover - [benchmarking/zeth-prover-bench/](../benchmarking/zeth-prover-bench/)

## How to Run

### Fetching Block Data
```bash
BLOCK_NUMBER=23791194
cd benchmarking
./fetch_block.sh $BLOCK_NUMBER
```

### Individual Provers

#### Erigon Block-Prover
```bash
bins/block-prove \
    --datadir $ERIGON_DATADIR \
    --block $BLOCK_NUMBER \
    --stark-proof
```

#### Ethrex
```bash
cd benchmarking/ethrex-prover-bench
cargo run --release -- \
    --witness "../witness-$BLOCK_NUMBER.json" \
    --block "../block-$BLOCK_NUMBER.json"
```

#### RSP
```bash
cd benchmarking/rsp-prover-bench
cargo run --release -- \
    --witness "../witness-$BLOCK_NUMBER.json" \
    --block "../block-$BLOCK_NUMBER.json"
```

#### Zeth
```bash
cd benchmarking/zeth-prover-bench
cargo run --release -- \
    --witness "../witness-$BLOCK_NUMBER.json" \
    --block "../block-$BLOCK_NUMBER.json"
```

## Metrics

### Erigon
- Total execution time
- Transpilation time (EVM -> risc-v)
- STARK proof generation time
- EVM instruction count
- ELF binary instruction count

### zkVM-based provers
- Total execution time
- Proof generation time
- CPU cycle count (VM overhead)

## Output Parsing
Use the provided Python scripts to parse benchmark outputs:
- [prover_bench_output.py](../benchmarking/prover_bench_output.py) - Parse zkVM prover outputs
- [erigon_block_output.py](../benchmarking/erigon_block_output.py) - Parse Erigon block-prover output

**TODO:** ideally we improve this to output the benchmark results directly into a table to be able to run benchmarks per commit etc.
