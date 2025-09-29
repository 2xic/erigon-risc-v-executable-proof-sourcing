package prover

import (
	"bytes"
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

func (cli *Cli) Execute(arg ...string) (string, error) {
	if len(arg) == 0 {
		return "", fmt.Errorf("no command provided")
	}

	cmd := exec.Command(arg[0], arg[1:]...)
	cmd.Dir = cli.workSpace

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	err := cmd.Run()
	return output.String(), err
}

func (cli *Cli) readFile(name string) ([]byte, error) {
	path := path.Join(cli.workSpace, name)
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return content, nil
}

type ProofGeneration struct {
	Proof  []byte
	AppVK  []byte
	Stdout string
}

type VerificationResult struct {
	Stdout string
	Valid  bool
}

type ResultsFile struct {
	AppVK string `json:"AppVK"`
	Proof string `json:"Proof"`
}

func (zkVm *ZkProver) Prove() (ProofGeneration, error) {
	cli, err := zkVm.SetupExecution()
	if err != nil {
		return ProofGeneration{}, NewZkProverError("failed to setup execution", err)
	}

	output, err := cli.Execute("cargo", "openvm", "prove", "app")
	if err != nil {
		return ProofGeneration{}, NewZkProverError("failed to execute prove command", err)
	}

	proof, err := cli.readFile("prover.app.proof")
	if err != nil {
		return ProofGeneration{}, err
	}
	appVk, err := cli.readFile("target/openvm/app.vk")
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

func (zkVm *ZkProver) StarkProve() (ProofGeneration, error) {
	workSpace, err := setupWorkspace([]byte(zkVm.content))
	if err != nil {
		return ProofGeneration{}, NewZkProverError("failed to setup workspace", err)
	}

	cli := NewCli(workSpace)
	_, err = cli.Execute("cargo", "openvm", "setup")
	if err != nil {
		return ProofGeneration{}, NewZkProverError("failed to execute prove command", err)
	}

	output, err := cli.Execute("cargo", "openvm", "prove", "stark")
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

func (zkVm *ZkProver) TestRun() (string, error) {
	cli, err := zkVm.SetupExecution()
	if err != nil {
		return "", err
	}

	output, err := cli.Execute("cargo", "openvm", "run")
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

func VerifyFromResults(resultsPath string) (VerificationResult, error) {
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
	output, err := cli.Execute("cargo", "openvm", "verify", "app", "--app-vk", appVKPath, "--proof", proofPath)
	result := VerificationResult{
		Stdout: output,
		Valid:  err == nil,
	}

	if err != nil {
		return result, NewZkProverError("verification failed", err)
	}

	return result, nil
}

func (zkVm *ZkProver) SetupExecution() (*Cli, error) {
	workSpace, err := setupWorkspace([]byte(zkVm.content))
	if err != nil {
		return nil, NewZkProverError("failed to setup workspace", err)
	}

	cli := NewCli(workSpace)

	_, err = cli.Execute("cargo", "openvm", "build")
	if err != nil {
		return nil, NewZkProverError("failed to build project", err)
	}

	_, err = cli.Execute("cargo", "openvm", "keygen")
	if err != nil {
		return nil, NewZkProverError("failed to generate keys", err)
	}
	return &cli, nil
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
