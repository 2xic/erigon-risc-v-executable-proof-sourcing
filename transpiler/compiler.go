package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

type AssemblyFile struct {
	content string
}

func (f *AssemblyFile) assembleToBytecode() ([]byte, error) {
	assembly := f.content
	fmt.Println(("==="))
	fmt.Println((assembly))
	fmt.Println(("==="))
	tmpFile, err := os.CreateTemp("", "*.s")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(assembly)
	if err != nil {
		return nil, err
	}
	tmpFile.Close()

	objFile := tmpFile.Name() + ".o"
	cmd := exec.Command("riscv64-linux-gnu-as", "-o", objFile, tmpFile.Name())
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to compile: %s", stderr.String())
	}
	defer os.Remove(objFile)

	binFile := tmpFile.Name() + ".bin"
	cmd = exec.Command("riscv64-linux-gnu-objcopy", "-O", "binary", objFile, binFile)
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	defer os.Remove(binFile)

	return os.ReadFile(binFile)
}
