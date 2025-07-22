package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
)

type AssemblyFile struct {
	content string
}

func (f *AssemblyFile) assembleToBytecode() ([]byte, error) {
	assembly := f.content
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
