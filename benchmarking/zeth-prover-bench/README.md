## Zeth 

### Install
```bash
curl -L https://risczero.com/install | bash
rzup install rust
rzup install cpp

cargo build --release  
```

### Running
See [readme](./README.md) in parent directory for how to get the input JSON fields.

```bash
export RISC0_PROVER=local 
cargo run --release -- \
    --witness "../witness-$BLOCK_NUMBER.json" \
    --block "../block-$BLOCK_NUMBER.json"
````
