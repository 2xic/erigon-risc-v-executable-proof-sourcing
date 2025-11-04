## Zeth 

### Install
```bash
curl -L https://risczero.com/install | bash
rzup install rust
rzup install cpp

cargo build --release  
```

### Running
```bash
export RISC0_PROVER=local 
cargo run --release -- -w ../reth-witness-550-sorted.json -b ../block_23174550.json --chain-id 1
````
