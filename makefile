.PHONY: lint lint-go lint-rust format-go

lint: lint-go lint-rust
	echo "done"

lint-go:
	@which goimports > /dev/null || go install golang.org/x/tools/cmd/goimports@latest
	cd transpiler && golangci-lint run --config ../golangci.yml
	cd transpiler && gofmt -w .
	cd transpiler && goimports -w .

lint-rust:
	cd prover/openvm && cargo clippy --all-targets --all-features -- -D warnings
	cd prover/openvm && cargo fmt

test:
	cd transpiler && go test -v ./...

remove_go_cache:
	rm -rf ~/.cache/go-build

# debug code for experiment with embedding Rust library for the unicorn test
emit-asm-prover:
#	cd prover/openvm && RUSTFLAGS="--emit=asm,obj" cargo openvm build --no-transpile
	cd prover/openvm/bigint && RUSTFLAGS="--emit=asm -C debuginfo=0" cargo +nightly-2025-02-14 build --target riscv32im-unknown-none-elf  -Zbuild-std=core,alloc
