# CLI Tool

Currently the CLI is using a mock state and we created this CLI to allow some testing.

## Build

```bash
make build
```

## Usage

```bash
./bins/prove -b <bytecode> [-c <calldata>] [-o <output>]
```

**Arguments:**
- `-b, --bytecode` (required): Contract bytecode (hex)
- `-c, --calldata` (optional): Call data (hex) 
- `-o, --output` (optional): Output prefix (default: "test.proof")

## Example

```bash
./bins/prove -b 608060...50033 -c 2e64cec1 -o counter_proof
```

Outputs `counter_proof.proof` and `counter_proof.vk` files, plus verification command.