.PHONY: lint lint-go lint-rust format-go build clean

lint: lint-go lint-rust
	echo "done"

build: bins/evm-prove bins/tx-prove

bins/evm-prove: cmd/evm-prove/main.go
	@mkdir -p bins
	go build -o bins/evm-prove ./cmd/prove

bins/tx-prove: cmd/tx-prove/main.go
	@mkdir -p bins
	go build -p 4 -o bins/tx-prove ./cmd/tx-prove

clean:
	rm -rf bins

lint-go:
	@which goimports > /dev/null || go install golang.org/x/tools/cmd/goimports@latest
	cd transpiler && golangci-lint run --config ../golangci.yml
	cd transpiler && gofmt -w .
	cd transpiler && goimports -w .

lint-rust:
	cd prover/openvm && cargo clippy --all-targets --all-features -- -D warnings
	cd prover/openvm && cargo fmt

test:
	cd transpiler && go test -parallel=1 -timeout 300s -v ./...

single_test:
		cd transpiler && go test -timeout 30s -run ^TestPushOpcodes$ erigon-transpiler-risc-v/transpiler

remove_go_cache:
	rm -rf ~/.cache/go-build

# debug code for experiment with embedding Rust library for the unicorn test
emit-asm-prover:
#	cd prover/openvm && RUSTFLAGS="--emit=asm,obj" cargo openvm build --no-transpile
	cd prover/openvm/bigint && RUSTFLAGS="--emit=asm -C debuginfo=0" cargo +nightly-2025-02-14 build --target riscv32im-unknown-none-elf  -Zbuild-std=core,alloc

counter_bytecode:
	solc --bin-runtime --via-ir --optimize contracts/Counter.sol

