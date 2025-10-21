import glob
import json

for i in glob.glob("block_*.json"):
    with open(i, "r") as file:
        data = json.load(file)
        transaction_count = data["transaction_count"]
        total_instructions = data["total_instructions"]
        proof_time_ms = data["proof_time_ms"]
        assembly_time_ms = data["assembly_time_ms"]
        total_time_ms = data["total_time_ms"]
        print(f"File: {i}")
        print(f"Transaction_count: {transaction_count}")
        print(f"Total_instructions: {total_instructions}")
        print(f"assembly_time_ms: {assembly_time_ms}")
        print(f"Proof_time_ms: {proof_time_ms}")
        print(f"total_time_ms: {total_time_ms}")
        print("===")
