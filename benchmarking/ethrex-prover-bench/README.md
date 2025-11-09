# Ethrex Prover Benchmark

## Setup SP1
```bash
curl https://sh.rustup.rs -sSf | sh
curl -L https://sp1up.succinct.xyz | bash
# sp1up --version 5.0.8
sp1up
```

### For risc0 - unused currently
```bash
curl -L https://risczero.com/install | bash  
~/.risc0/bin/rzup install cargo-risczero 3.0.3  
~/.risc0/bin/rzup install risc0-groth16  
~/.risc0/bin/rzup install rust
```

## Building
Building the benchmarking tool
```bash
sudo apt update 
sudo apt install -y pkg-config libssl-dev clang libclang-dev build-essential cmake
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