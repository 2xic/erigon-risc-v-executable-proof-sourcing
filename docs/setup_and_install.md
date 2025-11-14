## Setup and install

```bash
git submodule update --init --recursive
```

### Install golang, rust and openvm
Install [go toolchain](https://go.dev/doc/manage-install) and [Rust toolchain](https://doc.rust-lang.org/cargo/getting-started/installation.html)

Then to install OpenVM.
```bash
cargo +1.86 install --locked --git https://github.com/openvm-org/openvm.git --tag v1.4.0 cargo-openvm
```

### Install Unicorn
```bash
git clone https://github.com/unicorn-engine/unicorn.git
cd unicorn
git checkout f8c6db950420d2498700245269d0b647697c5666
mkdir build && cd build
cmake .. -DCMAKE_BUILD_TYPE=Release
make -j$(nproc)
make install
sudo ldconfig
```

### Building
```bash
make bins
```
