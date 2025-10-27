use anyhow::{anyhow, Result};
use chrono::Utc;
use clap::Parser;
use revm::state::Bytecode;
use rsp_client_executor::io::EthClientExecutorInput;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use sp1_sdk::{ProverClient, SP1Stdin};
use std::fs;
use std::path::PathBuf;
use std::time::Instant;
use tracing::info;

#[derive(Parser, Debug)]
#[command(author, version, about, long_about = None)]
struct Args {
    /// Path to the JSON file containing the execution witness
    #[arg(short = 'w', long)]
    witness_file: PathBuf,

    /// Path to the JSON file containing the block data
    #[arg(short = 'b', long)]
    block_file: PathBuf,

    /// Chain ID (default: 1 for Ethereum mainnet)
    #[arg(long, default_value = "1")]
    chain_id: u64,

    /// Output file for benchmark results
    #[arg(short, long)]
    output: Option<PathBuf>,
}

#[derive(Serialize, Deserialize, Debug)]
struct BenchmarkResult {
    chain_id: u64,
    backend: String,
    load_time_ms: u64,
    preparation_time_ms: u64,
    execution_time_ms: u64,
    proof_time_ms: Option<u64>,
    total_time_ms: u64,
    timestamp: String,
}

#[tokio::main]
async fn main() -> Result<()> {
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| tracing_subscriber::EnvFilter::new("info")),
        )
        .init();

    let args = Args::parse();

    info!("RSP Prover Benchmark");
    info!("Witness file: {:?}", args.witness_file);
    info!("Block file: {:?}", args.block_file);
    info!("Chain ID: {}", args.chain_id);

    let load_start = Instant::now();

    info!("Loading execution witness from file...");
    let witness_json = fs::read_to_string(&args.witness_file)
        .map_err(|e| anyhow!("Failed to read witness file: {}", e))?;

    info!("Loading block data from file...");
    let block_json = fs::read_to_string(&args.block_file)
        .map_err(|e| anyhow!("Failed to read block file: {}", e))?;

    let load_duration = load_start.elapsed();
    info!("Loaded files in {:?}", load_duration);

    let prep_start = Instant::now();

    info!("Parsing execution witness...");
    let witness_json_value: Value = serde_json::from_str(&witness_json)
        .map_err(|e| anyhow!("Failed to parse witness JSON: {}", e))?;

    let witness_data = if let Some(result) = witness_json_value.get("result") {
        result.clone()
    } else {
        witness_json_value
    };

    info!("Parsing block data...");
    let block_json_value: Value = serde_json::from_str(&block_json)
        .map_err(|e| anyhow!("Failed to parse block JSON: {}", e))?;

    let block_data = if let Some(result) = block_json_value.get("result") {
        result.clone()
    } else {
        block_json_value
    };

    let block_number = block_data.get("number")
        .and_then(|n| n.as_str())
        .ok_or_else(|| anyhow!("Block JSON missing 'number' field"))?;
    let block_number = u64::from_str_radix(block_number.trim_start_matches("0x"), 16)
        .map_err(|e| anyhow!("Failed to parse block number: {}", e))?;

    info!("Block number: {}", block_number);

    let genesis: rsp_primitives::genesis::Genesis = args.chain_id.try_into()?;

    info!("Constructing client executor input...");

    let execution_witness: alloy_rpc_types_debug::ExecutionWitness =
        serde_json::from_value(witness_data)
            .map_err(|e| anyhow!("Failed to deserialize execution witness: {}", e))?;

    info!("Building state from execution witness...");

    let mut ancestor_headers = Vec::new();
    for header_bytes in &execution_witness.headers {
        let header: alloy_consensus::Header = alloy_rlp::Decodable::decode(&mut header_bytes.as_ref())
            .map_err(|e| anyhow!("Failed to decode header: {}", e))?;
        ancestor_headers.push(header);
    }

    let parent_state_root = ancestor_headers
        .get(0)
        .ok_or_else(|| anyhow!("No headers in execution witness"))?
        .state_root;

    let parent_state = rsp_mpt::EthereumState::from_execution_witness(
        &execution_witness,
        parent_state_root,
    );

    let rpc_block: alloy_rpc_types::Block = serde_json::from_value(block_data.clone())
        .map_err(|e| anyhow!("Failed to deserialize RPC block: {}", e))?;

    let consensus_block: alloy_consensus::Block<alloy_consensus::TxEnvelope> = rpc_block.try_into().unwrap();

    let current_block = alloy_consensus::Block {
        header: consensus_block.header,
        body: alloy_consensus::BlockBody {
            transactions: consensus_block.body.transactions.into_iter().map(|tx| {
                let tx_json = serde_json::to_value(&tx).unwrap();
                serde_json::from_value(tx_json).unwrap()
            }).collect(),
            ommers: consensus_block.body.ommers,
            withdrawals: consensus_block.body.withdrawals,
        },
    };

    let bytecodes = execution_witness
        .codes
        .into_iter()
        .map(|code| Bytecode::new_raw(code.into()))
        .collect();

    let client_input = EthClientExecutorInput {
        current_block,
        ancestor_headers,
        parent_state,
        bytecodes,
        genesis,
        custom_beneficiary: None,
        opcode_tracking: false,
    };

    let prep_duration = prep_start.elapsed();
    info!("Prepared input in {:?}", prep_duration);

    info!("Setting up SP1 prover client...");
    let prover_client = ProverClient::from_env();

    // TODO: this should be simplified ... 
    // we should have a standard location for the ELF file and it should be automatically built
    let elf_path = PathBuf::from("../rsp/bin/client/target/elf-compilation/riscv32im-succinct-zkvm-elf/release/rsp-client");
    info!("Loading ELF from: {:?}", elf_path);
    let client_elf = fs::read(&elf_path)
        .map_err(|e| anyhow!("Failed to read ELF file at {:?}: {}. Please build it first with: cd benchmarking/rsp && cargo build --release --bin rsp-client --target riscv32im-succinct-zkvm-elf", elf_path, e))?;

    info!("Setting up proving key...");
    let (pk, _vk) = prover_client.setup(&client_elf);

    info!("Generating proof with SP1...");

    let input_bytes = bincode::serialize(&client_input)
        .map_err(|e| anyhow!("Failed to serialize input: {}", e))?;

    let mut stdin = SP1Stdin::new();
    stdin.write_vec(input_bytes);

    let exec_start = Instant::now();
    let (_, report) = prover_client.execute(&client_elf, &stdin).run()?;
    let exec_duration = exec_start.elapsed();

    info!("Execution completed in {:?}", exec_duration);
    info!("Total cycles: {}", report.total_instruction_count());

    info!("Generating compressed proof...");
    let prove_start = Instant::now();
    let proof = prover_client.prove(&pk, &stdin)
        .compressed()
        .run()?;
    let prove_duration = prove_start.elapsed();
    info!("Proof generated in {:?}", prove_duration);

    let total_duration = load_duration + prep_duration + prove_duration;

    let result = BenchmarkResult {
        chain_id: args.chain_id,
        backend: "SP1".to_string(),
        load_time_ms: load_duration.as_millis() as u64,
        preparation_time_ms: prep_duration.as_millis() as u64,
        execution_time_ms: exec_duration.as_millis() as u64,
        proof_time_ms: Some(prove_duration.as_millis() as u64),
        total_time_ms: total_duration.as_millis() as u64,
        timestamp: Utc::now().to_rfc3339(),
    };

    write_benchmark_results(&args, result, block_number)?;

    info!("Proof verification would happen here");
    info!("Proof size: {} bytes", proof.bytes().len());

    Ok(())
}

fn write_benchmark_results(args: &Args, result: BenchmarkResult, block_number: u64) -> Result<()> {
    let output_path = args.output.clone().unwrap_or_else(|| {
        PathBuf::from(format!("rsp_bench_{}.json", block_number))
    });

    let json_output = serde_json::to_string_pretty(&result)
        .map_err(|e| anyhow!("Failed to serialize benchmark result: {}", e))?;

    fs::write(&output_path, json_output)
        .map_err(|e| anyhow!("Failed to write output file: {}", e))?;

    info!("=== Benchmark Summary ===");
    info!("Backend: {}", result.backend);
    info!("Chain ID: {}", result.chain_id);
    info!("Load time: {} ms", result.load_time_ms);
    info!("Preparation time: {} ms", result.preparation_time_ms);
    info!("Execution time: {} ms", result.execution_time_ms);
    if let Some(proof_time) = result.proof_time_ms {
        info!("Proof time: {} ms", proof_time);
    }
    info!("Total time: {} ms", result.total_time_ms);
    info!("Results written to: {}", output_path.display());

    Ok(())
}
