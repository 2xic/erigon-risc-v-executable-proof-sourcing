# Ethrex Prover Benchmark

## Setup
```bash
curl -L https://sp1up.succinct.xyz | bash
sp1up --version 5.0.8
```

## Building
Building
```bash
cargo build --release --features sp1
```

## Data
Currently manually fetched the witness json from this PR
- https://etherscan.io/block/23174550
- https://github.com/ethereum/go-ethereum/pull/32216

## Usage
```bash
./target/release/ethrex-prover-bench \
  --rpc-url http://localhost:8545 \
  --block-number 1000000 \
  --prove
```