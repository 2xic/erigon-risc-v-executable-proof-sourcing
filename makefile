.PHONY: lint lint-go lint-rust

lint: lint-go lint-rust
	echo "done"

lint-go:
	cd transpiler && golangci-lint run

lint-rust:
	cd prover/openvm && cargo clippy --all-targets --all-features -- -D warnings
	cd prover/openvm && cargo fmt

test:
	cd transpiler && go test -v ./...

remove_go_cache:
	rm -rf ~/.cache/go-build
