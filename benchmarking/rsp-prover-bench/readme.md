## Setup 
```bash
sudo apt-get install m4

curl https://sh.rustup.rs -sSf | sh
sudo apt install build-essential -y 
curl -L https://sp1.succinct.xyz | bash
source /root/.bashrc
sp1up
```

## Build the RSP
```bash
cargo build ---release 
```

## To build the guest image
```bash
cargo +succinct build --release \
  --bin my_guest \
  --target riscv32im-succinct-zkvm-elf
```
