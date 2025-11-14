## Setup 
```bash
sudo apt-get install m4

curl https://sh.rustup.rs -sSf | sh
sudo apt install build-essential -y 
curl -L https://sp1.succinct.xyz | bash
source /root/.bashrc
sp1up
```

## Usage
See [readme](./README.md) in parent directory for how to get the input JSON fields.

```bash
cargo run --release -- \
    --witness "../witness-$BLOCK_NUMBER.json" \
    --block "../block-$BLOCK_NUMBER.json"
```
