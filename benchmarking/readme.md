## Benchmarking code
- [ethrex-prover-bench](./ethrex-prover-bench/README.md)
- [rsp-prover-bench](./rsp-prover-bench/README.md)
- [zeth](./zeth-prover-bench/README.md)

## Enable cycle counting for ethrex 
```bash
cd ethrex && git apply ../ethrex-cycle-counting.patch
```

## Getting data
Block data:
```bash
curl -X POST https://your-rpc-url-here \
-H "Content-Type: application/json" \
-d '{
    "jsonrpc": "2.0",
    "method": "eth_getBlockByNumber",
    "params": ["0x1619d96", true],
    "id": 1
}' > block_23174550.js
```

Execution witness:
```bash
curl -X POST https://your-rpc-url-here \
-H "Content-Type: application/json" \
-d '{
    "jsonrpc": "2.0",
    "method": "debug_executionWitness",
    "params": ["0x15dbf23"],
    "id": 1
}'
```
