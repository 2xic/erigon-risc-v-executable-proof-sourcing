package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

type AssemblyFile struct {
	instructions []Instruction
}

func (a *AssemblyFile) toDebugFile() string {
	return a.toFile(true)
}

func (a *AssemblyFile) toZkFile() string {
	return a.toFile(false)
}

func (a *AssemblyFile) toFile(withEbreak bool) string {
	instructions := make([]string, 0)
	for i := range a.instructions {
		if !withEbreak && a.instructions[i].name == "EBREAK" {
			continue
		}
		instructions = append(instructions, fmt.Sprintf("%s %s", a.instructions[i].name, strings.Join(a.instructions[i].operands, ", ")))
	}

	content := strings.Join(instructions, "\n")
	return content
}

func (f *AssemblyFile) toBytecode() ([]byte, error) {
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

func (f *AssemblyFile) toToolChainCompatibleAssembly() (string, error) {
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
