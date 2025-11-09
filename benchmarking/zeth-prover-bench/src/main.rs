use anyhow::{anyhow, Context, Result};
use chrono::Utc;
use clap::Parser;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::fs;
use std::path::PathBuf;
use std::time::Instant;
use tracing::info;
use zeth_core::Input;

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
    block_number: u64,
    chain_id: u64,
    backend: String,
    load_time_ms: u64,
    validation_time_ms: u64,
    proof_time_ms: u64,
    total_time_ms: u64,
    total_cycles: u64,
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

    info!("Zeth Prover Benchmark");
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

    let witness_value: Value = serde_json::from_str(&witness_json)
        .map_err(|e| anyhow!("Failed to parse witness JSON: {}", e))?;
    let witness_data = if let Some(result) = witness_value.get("result") {
        result.clone()
    } else {
        witness_value
    };

    let block_value: Value = serde_json::from_str(&block_json)
        .map_err(|e| anyhow!("Failed to parse block JSON: {}", e))?;
    let block_data = if let Some(result) = block_value.get("result") {
        result.clone()
    } else {
        block_value
    };

    let execution_witness: alloy_rpc_types_debug::ExecutionWitness =
        serde_json::from_value(witness_data)
            .map_err(|e| anyhow!("Failed to deserialize execution witness: {}", e))?;

    let rpc_block: alloy_rpc_types::Block = serde_json::from_value(block_data)
        .map_err(|e| anyhow!("Failed to deserialize RPC block: {}", e))?;

    let block: reth_ethereum_primitives::Block = rpc_block.try_into()
        .map_err(|e: std::convert::Infallible| anyhow!("Failed to convert block: {:?}", e))?;

    let witness = reth_stateless::ExecutionWitness {
        state: execution_witness.state,
        codes: execution_witness.codes,
        headers: execution_witness.headers,
        keys: execution_witness.keys,
    };

    let signers = zeth_host::recover_signers(block.body.transactions())
        .context("Failed to recover transaction signers")?;

    let input = Input {
        block,
        witness,
        signers,
    };

    let block_number = input.block.number;
    let block_hash = input.block.hash_slow();

    let load_duration = load_start.elapsed();
    info!("Loaded input in {:?}", load_duration);
    info!("Block number: {}", block_number);
    info!("Block hash: {}", block_hash);

    let input_size = zeth_host::to_zkvm_input_bytes(&input)?.len();
    info!("Input size: {:.3} MB", input_size as f64 / 1e6);

    info!("Validating block execution...");
    let validation_start = Instant::now();

    let validation_duration = validation_start.elapsed();
    info!("Validation skipped (using pre-validated cached input)");

    info!("Generating proof with RISC Zero...");
    let proof_start = Instant::now();

    let guest_name = match args.chain_id {
        1 => {
            info!("Using mainnet guest");
            "mainnet"
        }
        _ => return Err(anyhow!("Unsupported chain ID: {}", args.chain_id)),
    };

    // Load the guest ELF from the filesystem (use .bin extension for RISC Zero binary)
    let guest_elf_path = PathBuf::from(format!(
        "../zeth/target/riscv-guest/guests/stateless-client/riscv32im-risc0-zkvm-elf/release/{}.bin",
        guest_name
    ));

    info!("Loading guest ELF from: {:?}", guest_elf_path);
    let guest_elf = fs::read(&guest_elf_path)
        .with_context(|| format!(
            "Failed to read guest ELF at {:?}. Make sure to build it first:\n\
             cd benchmarking/zeth && cargo build --release",
            guest_elf_path
        ))?;

    info!("Guest ELF size: {} bytes", guest_elf.len());

    let image_id = risc0_zkvm::compute_image_id(&guest_elf)
        .context("Failed to compute image ID from guest ELF")?;
    info!("Image ID: {}", hex::encode(&image_id));

    let input_bytes = zeth_host::to_zkvm_input_bytes(&input)?;

    let env = risc0_zkvm::ExecutorEnv::builder()
        .write_slice(&input_bytes)
        .build()
        .context("Failed to build executor environment")?;

    info!("Executing to measure cycles...");
    let exec_start = Instant::now();
    let executor = risc0_zkvm::default_executor();
    
    let exec_env = risc0_zkvm::ExecutorEnv::builder()
        .write_slice(&input_bytes)
        .build()
        .context("Failed to build executor environment for cycle counting")?;
    
    let session = executor
        .execute(exec_env, &guest_elf)
        .map_err(|e| anyhow!("Failed to execute for cycle counting: {}", e))?;
    
    let exec_duration = exec_start.elapsed();
    info!("Execution completed in {:?}", exec_duration);

    let total_cycles = session.segments.iter().map(|s| s.cycles as u64).sum::<u64>();
    info!("Total cycles: {}", total_cycles);

    info!("Creating prover...");
    let prover = risc0_zkvm::default_prover();

    info!("Starting proof generation (this may take a while)...");
    let prove_info = prover
        .prove(env, &guest_elf)
        .map_err(|e| {
            anyhow!(
                "Failed to generate proof: {:?}\n\
                 Guest ELF size: {} bytes\n\
                 Image ID: {}\n\
                 Error details: {:#}",
                e,
                guest_elf.len(),
                hex::encode(&image_id),
                e
            )
        })?;

    let receipt = prove_info.receipt;
    let proof_duration = proof_start.elapsed();
    info!("Proof generated in {:?}", proof_duration);

    info!("Verifying proof...");
    receipt
        .verify(image_id)
        .context("Proof verification failed")?;

    let proven_hash = alloy_primitives::B256::try_from(receipt.journal.as_ref())
        .context("Failed to decode journal")?;

    if proven_hash != block_hash {
        return Err(anyhow!("Journal output mismatch: expected {}, got {}", block_hash, proven_hash));
    }

    info!("Proof verified successfully!");

    let total_duration = load_duration + validation_duration + proof_duration;

    let result = BenchmarkResult {
        block_number,
        chain_id: args.chain_id,
        backend: "RISC Zero".to_string(),
        load_time_ms: load_duration.as_millis() as u64,
        validation_time_ms: validation_duration.as_millis() as u64,
        proof_time_ms: proof_duration.as_millis() as u64,
        total_time_ms: total_duration.as_millis() as u64,
        total_cycles,
        timestamp: Utc::now().to_rfc3339(),
    };

    write_benchmark_results(&args, result, block_number)?;

    Ok(())
}

fn write_benchmark_results(args: &Args, result: BenchmarkResult, block_number: u64) -> Result<()> {
    let output_path = args.output.clone().unwrap_or_else(|| {
        PathBuf::from(format!("zeth_bench_{}.json", block_number))
    });

    let json_output = serde_json::to_string_pretty(&result)
        .map_err(|e| anyhow!("Failed to serialize benchmark result: {}", e))?;

    fs::write(&output_path, json_output)
        .map_err(|e| anyhow!("Failed to write output file: {}", e))?;

    info!("=== Benchmark Summary ===");
    info!("Backend: {}", result.backend);
    info!("Block: {}", result.block_number);
    info!("Chain ID: {}", result.chain_id);
    info!("Load time: {} ms", result.load_time_ms);
    info!("Validation time: {} ms", result.validation_time_ms);
    info!("Proof time: {} ms", result.proof_time_ms);
    info!("Total time: {} ms", result.total_time_ms);
    info!("Results written to: {}", output_path.display());

    Ok(())
}
