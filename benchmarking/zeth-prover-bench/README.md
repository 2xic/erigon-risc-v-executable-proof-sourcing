## Zeth 

### Install
```bash
curl -L https://risczero.com/install | bash
rzup install rust
rzup install cpp

export RISC0_SKIP_BUILD=1  
cargo build --release  
```

### Running
```bash
cargo run --release -- -w ../reth-witness-550-sorted.json -b ../block_23174550.json --chain-id 1
````
