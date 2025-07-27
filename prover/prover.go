package prover

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
)

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

func (zkVm *ZkProver) Prove() (string, error) {
	cli, err := zkVm.SetupExecution()
	if err != nil {
		return "", err
	}

	output, err := cli.Execute("cargo", "openvm", "prove", "app", "--input", "0x010A00000000000000")
	if err != nil {
		return "", err
	}

	return string(output), nil
}

func (zkVm *ZkProver) TestRun() (string, error) {
	cli, err := zkVm.SetupExecution()
	if err != nil {
		return "", err
	}

	output, err := cli.Execute("cargo", "openvm", "run", "--input", "0x010A00000000000000")
	if err != nil {
		return "", err
	}

	return string(output), nil
}

func (zkVm *ZkProver) SetupExecution() (*Cli, error) {
	workSpace, err := setupWorkspace([]byte(zkVm.content))
	if err != nil {
		return nil, err
	}

	cli := NewCli(workSpace)

	_, err = cli.Execute("cargo", "openvm", "build")
	if err != nil {
		return nil, err
	}

	_, err = cli.Execute("cargo", "openvm", "keygen")
	if err != nil {
		return nil, err
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
