package prover

import (
	"bytes"
	"context"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"
)

type ZkProverError struct {
	Message    string
	Underlying error
}

func NewZkProverError(message string, underlying error) *ZkProverError {
	return &ZkProverError{
		Message:    message,
		Underlying: underlying,
	}
}

func (e *ZkProverError) Error() string {
	return fmt.Sprintf("ZkProver error: %s: %v", e.Message, e.Underlying)
}

//go:embed openvm/*
var zkVMToolchain embed.FS

type ZkProver struct {
	content string
}

func NewZkProver(content string) *ZkProver {
	return &ZkProver{
		content: content,
	}
}

type Cli struct {
	workSpace string
}

func NewCli(workSpace string) Cli {
	return Cli{
		workSpace: workSpace,
	}
}

func (cli *Cli) Execute(ctx context.Context, arg ...string) (string, error) {
	if len(arg) == 0 {
		return "", fmt.Errorf("no command provided")
	}

	cmd := exec.CommandContext(ctx, arg[0], arg[1:]...)
	cmd.Dir = cli.workSpace

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil && stderr.Len() > 0 {
		return stdout.String(), fmt.Errorf("%w: stderr: %s", err, stderr.String())
	}
	return stdout.String(), err
}

func (cli *Cli) readFile(name string) ([]byte, error) {
	path := path.Join(cli.workSpace, name)
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return content, nil
}

type ProofTiming struct {
	BuildTimeMs  int64
	KeygenTimeMs int64
	SetupTimeMs  int64
	ProveTimeMs  int64
	ReadTimeMs   int64
	TotalTimeMs  int64
}

type ProofGeneration struct {
	Proof                 []byte
	AppVK                 []byte
	Stdout                string
	Timing                ProofTiming
	EstimatedInstructions int64
}

type VerificationResult struct {
	Stdout string
	Valid  bool
}

type ResultsFile struct {
	AppVK string `json:"AppVK"`
	Proof string `json:"Proof"`
}

func (zkVm *ZkProver) Prove(ctx context.Context) (ProofGeneration, error) {
	setupStart := time.Now()
	cli, setupTiming, err := zkVm.SetupExecution(ctx)
	if err != nil {
		return ProofGeneration{}, NewZkProverError("failed to setup execution", err)
	}
	setupTime := time.Since(setupStart)

	proveStart := time.Now()
	output, err := cli.Execute(ctx, "cargo", "openvm", "prove", "app")
	if err != nil {
		return ProofGeneration{}, NewZkProverError("failed to execute prove command", err)
	}
	proveTime := time.Since(proveStart)

	readStart := time.Now()
	proof, err := cli.readFile("prover.app.proof")
	if err != nil {
		return ProofGeneration{}, err
	}
	appVk, err := cli.readFile("target/openvm/app.vk")
	if err != nil {
		return ProofGeneration{}, err
	}
	estimatedInstructions := zkVm.getEstimatedInstructionCount(cli)

	readTime := time.Since(readStart)

	totalTime := setupTime + proveTime + readTime

	results := ProofGeneration{
		Proof:  proof,
		AppVK:  appVk,
		Stdout: output,
		Timing: ProofTiming{
			BuildTimeMs:  setupTiming.BuildTimeMs,
			KeygenTimeMs: setupTiming.KeygenTimeMs,
			SetupTimeMs:  setupTime.Milliseconds(),
			ProveTimeMs:  proveTime.Milliseconds(),
			ReadTimeMs:   readTime.Milliseconds(),
			TotalTimeMs:  totalTime.Milliseconds(),
		},
		EstimatedInstructions: estimatedInstructions,
	}

	return results, nil
}

func (zkVm *ZkProver) StarkProve(ctx context.Context) (ProofGeneration, error) {
	workSpace, err := setupWorkspace([]byte(zkVm.content))
	if err != nil {
		return ProofGeneration{}, NewZkProverError("failed to setup workspace", err)
	}

	cli := NewCli(workSpace)
	_, err = cli.Execute(ctx, "cargo", "openvm", "setup")
	if err != nil {
		return ProofGeneration{}, NewZkProverError("failed to execute prove command", err)
	}

	output, err := cli.Execute(ctx, "cargo", "openvm", "prove", "stark")
	if err != nil {
		return ProofGeneration{}, NewZkProverError("failed to execute prove command", err)
	}

	proof, err := cli.readFile("prover.stark.proof")
	if err != nil {
		return ProofGeneration{}, err
	}
	appVk, err := os.ReadFile("/root/.openvm/agg_stark.vk")
	if err != nil {
		return ProofGeneration{}, err
	}

	results := ProofGeneration{
		Proof:  proof,
		AppVK:  appVk,
		Stdout: output,
	}

	return results, nil
}

func (zkVm *ZkProver) TestRun(ctx context.Context) (string, error) {
	cli, _, err := zkVm.SetupExecution(ctx)
	if err != nil {
		return "", err
	}

	output, err := cli.Execute(ctx, "cargo", "openvm", "run")
	if err != nil {
		return "", err
	}

	executionOutput := ""
	for _, line := range bytes.Split([]byte(output), []byte("\n")) {
		if bytes.HasPrefix(line, []byte("Execution output:")) {
			executionOutput = string(line)
			break
		}
	}
	if executionOutput == "" {
		return "", fmt.Errorf("execution output not found in the output: %s", output)
	}

	return executionOutput, nil
}

func VerifyFromResults(ctx context.Context, resultsPath string) (VerificationResult, error) {
	if resultsPath == "" {
		resultsPath = "results.json"
	}

	resultsData, err := os.ReadFile(resultsPath)
	if err != nil {
		return VerificationResult{}, NewZkProverError("failed to read results file", err)
	}

	var results ResultsFile
	if err := json.Unmarshal(resultsData, &results); err != nil {
		return VerificationResult{}, NewZkProverError("failed to parse results file", err)
	}

	appVKBytes, err := hex.DecodeString(results.AppVK)
	if err != nil {
		return VerificationResult{}, NewZkProverError("failed to decode AppVK hex", err)
	}

	proofBytes, err := hex.DecodeString(results.Proof)
	if err != nil {
		return VerificationResult{}, NewZkProverError("failed to decode Proof hex", err)
	}

	tmpDir, err := os.MkdirTemp("", "openvm-verify-*")
	if err != nil {
		return VerificationResult{}, NewZkProverError("failed to create temp directory", err)
	}
	defer os.RemoveAll(tmpDir)

	appVKPath := filepath.Join(tmpDir, "app.vk")
	if err := os.WriteFile(appVKPath, appVKBytes, 0644); err != nil {
		return VerificationResult{}, NewZkProverError("failed to write app.vk file", err)
	}

	proofPath := filepath.Join(tmpDir, "proof.app.proof")
	if err := os.WriteFile(proofPath, proofBytes, 0644); err != nil {
		return VerificationResult{}, NewZkProverError("failed to write proof file", err)
	}

	cli := NewCli(tmpDir)
	output, err := cli.Execute(ctx, "cargo", "openvm", "verify", "app", "--app-vk", appVKPath, "--proof", proofPath)
	result := VerificationResult{
		Stdout: output,
		Valid:  err == nil,
	}

	if err != nil {
		return result, NewZkProverError("verification failed", err)
	}

	return result, nil
}

type SetupTiming struct {
	BuildTimeMs  int64
	KeygenTimeMs int64
}

func (zkVm *ZkProver) SetupExecution(ctx context.Context) (*Cli, SetupTiming, error) {
	workSpace, err := setupWorkspace([]byte(zkVm.content))
	if err != nil {
		return nil, SetupTiming{}, NewZkProverError("failed to setup workspace", err)
	}

	cli := NewCli(workSpace)

	buildStart := time.Now()
	_, err = cli.Execute(ctx, "cargo", "openvm", "build")
	if err != nil {
		return nil, SetupTiming{}, NewZkProverError("failed to build project", err)
	}
	buildTime := time.Since(buildStart)

	keygenStart := time.Now()
	_, err = cli.Execute(ctx, "cargo", "openvm", "keygen")
	if err != nil {
		return nil, SetupTiming{}, NewZkProverError("failed to generate keys", err)
	}
	keygenTime := time.Since(keygenStart)

	timing := SetupTiming{
		BuildTimeMs:  buildTime.Milliseconds(),
		KeygenTimeMs: keygenTime.Milliseconds(),
	}

	return &cli, timing, nil
}

func setupWorkspace(newRiscContent []byte) (string, error) {
	tmpDir, err := os.MkdirTemp("", "zkvm-toolchain-*")
	if err != nil {
		return "", err
	}

	if err := extractEmbedFS(zkVMToolchain, tmpDir); err != nil {
		return "", err
	}

	workspaceDirectory := path.Join(tmpDir, "openvm")

	riscPath := filepath.Join(workspaceDirectory, "src", "risc.asm")
	if err := os.WriteFile(riscPath, newRiscContent, 0644); err != nil {
		return "", fmt.Errorf("failed to write risc.asm: %w", err)
	}

	return workspaceDirectory, nil
}

func extractEmbedFS(fsys embed.FS, dstDir string) error {
	return fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || path == "." {
			return err
		}

		dstPath := filepath.Join(dstDir, path)

		if d.IsDir() {
			return os.MkdirAll(dstPath, 0755)
		}

		return extractFile(fsys, path, dstPath)
	})
}

func extractFile(fsys embed.FS, srcPath, dstPath string) error {
	src, err := fsys.Open(srcPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := src.Close(); err != nil {
			log.Printf("Failed close src %v", err)
		}
	}()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := dst.Close(); err != nil {
			log.Printf("Failed close dst %v", err)
		}
	}()
	_, err = io.Copy(dst, src)
	return err
}

func (zkVm *ZkProver) getEstimatedInstructionCount(cli *Cli) int64 {
	output, err := cli.Execute(context.Background(), "riscv64-unknown-elf-objdump", "-d", "target/riscv32im-risc0-zkvm-elf/release/prover")
	if err != nil {
		return 0
	}

	lines := bytes.Split([]byte(output), []byte("\n"))
	return int64(len(lines))
}
