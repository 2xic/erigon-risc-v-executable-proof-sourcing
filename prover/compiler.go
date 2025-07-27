package prover

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

type AssemblyFile struct {
	Instructions []Instruction
}

type Instruction struct {
	Name     string
	Operands []string
}

func (a *AssemblyFile) toDebugFile() string {
	return a.toFile(false)
}

func (a *AssemblyFile) toZkFile() string {
	return a.toFile(true)
}

func (a *AssemblyFile) toFile(skipEbreak bool) string {
	instructions := make([]string, 0)
	for i := range a.Instructions {
		if skipEbreak && a.Instructions[i].Name == "EBREAK" {
			continue
		}
		instructions = append(instructions, fmt.Sprintf("%s %s", a.Instructions[i].Name, strings.Join(a.Instructions[i].Operands, ", ")))
	}

	content := strings.Join(instructions, "\n")
	return content
}

func (f *AssemblyFile) ToBytecode() ([]byte, error) {
	assembly := f.toDebugFile()
	tmpFile, err := os.CreateTemp("", "*.s")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			log.Printf("Failed to remove temporary file %s: %v", tmpFile.Name(), err)
		}
	}()

	_, err = tmpFile.WriteString(assembly)
	if err != nil {
		return nil, err
	}
	err = tmpFile.Close()
	if err != nil {
		return nil, err
	}

	objFile := tmpFile.Name() + ".o"
	cmd := exec.Command("riscv64-linux-gnu-as", "-o", objFile, tmpFile.Name())
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to compile: %s", stderr.String())
	}
	defer func() {
		if err := os.Remove(objFile); err != nil {
			log.Printf("Failed to remove temporary file %s: %v", objFile, err)
		}
	}()

	binFile := tmpFile.Name() + ".bin"
	cmd = exec.Command("riscv64-linux-gnu-objcopy", "-O", "binary", objFile, binFile)
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	defer func() {
		if err := os.Remove(binFile); err != nil {
			log.Printf("Failed to remove temporary file %s: %v", binFile, err)
		}
	}()
	return os.ReadFile(binFile)
}

func (f *AssemblyFile) ToToolChainCompatibleAssembly() (string, error) {
	format := `
.global execute
execute:
	# Save stack
	mv s2, sp

	%s

	# Restore stack
	mv sp, s2
	ret	
	`
	return fmt.Sprintf(format, f.toZkFile()), nil
}
