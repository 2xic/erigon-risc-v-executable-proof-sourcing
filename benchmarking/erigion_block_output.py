import glob
import json

for i in glob.glob("block_*.json"):
    with open(i, "r") as file:
        data = json.load(file)
        block_fetch_time_ms = data["block_fetch_time_ms"]
        tx_fetch_time_ms = data["tx_fetch_time_ms"]
        transaction_count = data["transaction_count"]
        total_instructions = data["total_instructions"]
        transpile_time_ms = data["transpile_time_ms"]
        assembly_time_ms = data["assembly_time_ms"]
        proof_time_ms = data["proof_time_ms"]
        proof_build_time_ms = data["proof_build_time_ms"]
        proof_keygen_time_ms = data["proof_keygen_time_ms"]
        proof_setup_time_ms = data["proof_setup_time_ms"]
        proof_prove_time_ms = data["proof_prove_time_ms"]
        proof_read_time_ms = data["proof_read_time_ms"]
        total_time_ms = data["total_time_ms"]
        print(f"File: {i}")
        print(f"block_fetch_time_ms: {block_fetch_time_ms}")
        print(f"tx_fetch_time_ms: {tx_fetch_time_ms}")
        print(f"transaction_count: {transaction_count}")
        print(f"total_instructions: {total_instructions}")
        print(f"transpile_time_ms: {transpile_time_ms}")
        print(f"assembly_time_ms: {assembly_time_ms}")
        print(f"proof_time_ms: {proof_time_ms}")
        print(f"  proof_build_time_ms: {proof_build_time_ms}")
        print(f"  proof_keygen_time_ms: {proof_keygen_time_ms}")
        print(f"  proof_setup_time_ms: {proof_setup_time_ms}")
        print(f"  proof_prove_time_ms: {proof_prove_time_ms}")
        print(f"  proof_read_time_ms: {proof_read_time_ms}")
        print(f"total_time_ms: {total_time_ms}")
        print("===")
