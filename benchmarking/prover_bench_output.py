import glob
import json

# Handle ethrex benchmark results
ethrex_files = glob.glob("ethrex-prover-bench/*_ethrex.json")
if len(ethrex_files) > 0:
    print("=== ETHREX BENCHMARK RESULTS ===")
    for i in ethrex_files:
        with open(i, "r") as file:
            data = json.load(file)
            print(f"File: {i}")
            print(f"block_number: {data.get('block_number', 'N/A')}")
            print(f"backend: {data.get('backend', 'N/A')}")
            print(f"fetch_time_ms: {data.get('fetch_time_ms', 0)}")
            print(f"witness_time_ms: {data.get('witness_time_ms', 0)}")
            print(f"execution_time_ms: {data.get('execution_time_ms', 0)}")
            print(f"proof_time_ms: {data.get('proof_time_ms', 0)}")
            print(f"total_time_ms: {data.get('total_time_ms', 0)}")
            print(f"total_cycles: {data.get('total_cycles', 0)}")
            print(f"total_syscalls: {data.get('total_syscalls', 0)}")
            print("===")
else:
    print("No ethrex benchmark files found (expected: ethrex-prover-bench/*_ethrex.json)")

# Handle RSP benchmark results
rsp_files = glob.glob("rsp-prover-bench/rsp_bench_*.json")
if len(rsp_files) > 0:
    print("\n=== RSP BENCHMARK RESULTS ===")
    for i in rsp_files:
        with open(i, "r") as file:
            data = json.load(file)
            print(f"File: {i}")
            print(f"backend: {data.get('backend', 'N/A')}")
            print(f"load_time_ms: {data.get('load_time_ms', 0)}")
            print(f"preparation_time_ms: {data.get('preparation_time_ms', 0)}")
            print(f"execution_time_ms: {data.get('execution_time_ms', 0)}")
            proof_time = data.get('proof_time_ms')
            if proof_time is not None:
                print(f"proof_time_ms: {proof_time}")
            else:
                print("proof_time_ms: None")
            print(f"total_time_ms: {data.get('total_time_ms', 0)}")
            print(f"total_cycles: {data.get('total_cycles', 0)}")
            print(f"total_syscalls: {data.get('total_syscalls', 0)}")
            print("===")
else:
    print("No RSP benchmark files found (expected: rsp-prover-bench/rsp_bench_*.json)")

# Handle Zeth benchmark results  
zeth_files = glob.glob("zeth-prover-bench/zeth_bench_*.json")
if len(zeth_files) > 0:
    print("\n=== ZETH BENCHMARK RESULTS ===")
    for i in zeth_files:
        with open(i, "r") as file:
            data = json.load(file)
            print(f"File: {i}")
            print(f"block_number: {data.get('block_number', 'N/A')}")
            print(f"backend: {data.get('backend', 'N/A')}")
            print(f"load_time_ms: {data.get('load_time_ms', 0)}")
            print(f"validation_time_ms: {data.get('validation_time_ms', 0)}")
            print(f"proof_time_ms: {data.get('proof_time_ms', 0)}")
            print(f"total_time_ms: {data.get('total_time_ms', 0)}")
            print(f"total_cycles: {data.get('total_cycles', 0)}")
            print("===")
else:
    print("No Zeth benchmark files found (expected: zeth-prover-bench/zeth_bench_*.json)")