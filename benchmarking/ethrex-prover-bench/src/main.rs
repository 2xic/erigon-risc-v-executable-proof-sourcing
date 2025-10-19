use anyhow::{anyhow, Result};
use clap::Parser;
use ethrex_common::types::ChainConfig;
use ethrex_prover_lib::{execute, prove, to_batch_proof, backend::Backend};
use ethrex_rpc::{
    clients::eth::EthClient,
    debug::execution_witness::{execution_witness_from_rpc_chain_config, RpcExecutionWitness},
    types::block_identifier::BlockIdentifier,
};
use guest_program::input::ProgramInput;
use serde_json::Value;
use std::fs;
use std::path::PathBuf;
use std::time::Instant;
use tracing::info;

#[derive(Parser, Debug)]
#[command(author, version, about, long_about = None)]
struct Args {
    #[arg(short, long)]
    rpc_url: String,

    #[arg(short, long)]
    block_number: u64,

    #[arg(long)]
    end_block: Option<u64>,

    #[arg(long, default_value = "1")]
    chain_id: u64,

    #[arg(short, long)]
    prove: bool,

    #[arg(short = 'w', long)]
    witness_file: Option<PathBuf>,
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

    let backend = Backend::SP1;

    info!("Ethrex Prover Benchmark (L1 blocks)");
    info!("RPC URL: {}", args.rpc_url);
    info!("Block: {}", args.block_number);
    if let Some(end) = args.end_block {
        info!("End block: {}", end);
    }
    info!("Backend: {:?}", backend);
    info!("Mode: {}", if args.prove { "prove" } else { "execute" });

    let client = EthClient::new(&args.rpc_url)
        .map_err(|e| anyhow!("Failed to create RPC client: {}", e))?;

    info!("Fetching block {} from RPC...", args.block_number);
    let fetch_start = Instant::now();

    let block = client
        .get_raw_block(BlockIdentifier::Number(args.block_number))
        .await
        .map_err(|e| anyhow!("Failed to fetch block {}: {}", args.block_number, e))?;

    info!("Fetched block in {:?}", fetch_start.elapsed());

    let witness_start = Instant::now();
    let rpc_witness = if let Some(witness_path) = &args.witness_file {
        info!("Loading execution witness from file: {:?}", witness_path);
        let witness_json = fs::read_to_string(witness_path)
            .map_err(|e| anyhow!("Failed to read witness file: {}", e))?;
        let witness_data: Value = serde_json::from_str(&witness_json)
            .map_err(|e| anyhow!("Failed to parse witness JSON: {}", e))?;

        let result = witness_data.get("result")
            .ok_or_else(|| anyhow!("Witness JSON missing 'result' field"))?;

        serde_json::from_value(result.clone())
            .map_err(|e| anyhow!("Failed to deserialize witness data: {}", e))?
    } else {
        info!("Fetching execution witness from RPC...");
        client
            .get_witness(BlockIdentifier::Number(args.block_number), None)
            .await
            .map_err(|e| anyhow!("Failed to fetch execution witness: {}", e))?
    };

    info!("Got execution witness in {:?}", witness_start.elapsed());

    let mut chain_config = ChainConfig::default();
    chain_config.chain_id = args.chain_id;
    chain_config.shanghai_time = Some(1681338455);
    chain_config.cancun_time = Some(1710338135);
    chain_config.prague_time = Some(1746612311);
    let execution_witness = execution_witness_from_rpc_chain_config(
        rpc_witness,
        chain_config,
        args.block_number,
    )
    .map_err(|e| anyhow!("Failed to convert RPC witness: {}", e))?;

    let input = ProgramInput {
        blocks: vec![block],
        execution_witness,
        elasticity_multiplier: 2,
        fee_config: None,
    };

    if args.prove {
        info!("Generating proof with {:?} backend...", backend);
        let prove_start = Instant::now();

        let proof_output = prove(backend, input, false)
            .map_err(|e| anyhow!("Failed to generate proof: {}", e))?;

        let prove_duration = prove_start.elapsed();
        info!("✓ Proof generated in {:?}", prove_duration);

        let batch_proof = to_batch_proof(proof_output, false)
            .map_err(|e| anyhow!("Failed to convert to batch proof: {}", e))?;

        info!("Batch proof type: {:?}", batch_proof);

        info!("=== Benchmark Summary ===");
        info!("Backend: {:?}", backend);
        info!("Block: {}", args.block_number);
        info!("Proof generation time: {:?}", prove_duration);
    } else {
        info!("Executing with {:?} backend (no proof generation)...", backend);
        let exec_start = Instant::now();

        execute(backend, input)
            .map_err(|e| anyhow!("Failed to execute: {}", e))?;

        let exec_duration = exec_start.elapsed();
        info!("✓ Execution completed in {:?}", exec_duration);

        info!("=== Benchmark Summary ===");
        info!("Backend: {:?}", backend);
        info!("Block: {}", args.block_number);
        info!("Execution time: {:?}", exec_duration);
    }

    Ok(())
}
