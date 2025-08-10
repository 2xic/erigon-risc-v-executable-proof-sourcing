package prover

import (
	"bytes"
	_ "embed"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

const InstructionEBREAK = "EBREAK"

type AssemblyFile struct {
	Instructions []Instruction
}

//go:embed resources/lib.asm
var libFile []byte

type Instruction struct {
	Name     string
	Operands []string
}

func (a *AssemblyFile) toDebugFile() string {
	instructions := a.toFile(false)
	file := `
.section .text
.global execute
execute:
%s 
%s
	`
	content := fmt.Sprintf(file, instructions, string(libFile))
	return content
}

func (a *AssemblyFile) toZkFile() string {
	return a.toFile(true)
}

func (a *AssemblyFile) toFile(skipEbreak bool) string {
	instructions := make([]string, 0)
	for _, instr := range a.Instructions {
		if skipEbreak && instr.Name == InstructionEBREAK {
			continue
		}
		stringified := fmt.Sprintf("\t%s %s", instr.Name, strings.Join(instr.Operands, ", "))
		instructions = append(instructions, stringified)
	}

	content := strings.Join(instructions, "\n")
	return content
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

// Used by the testing setup
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
	cmd := exec.Command("riscv64-linux-gnu-gcc",
		"-march=rv32im",
		"-mabi=ilp32",
		"-nostdlib",      // No standard library
		"-static",        // Static linking
		"-Wl,-Ttext=0x0", // Set text address
		"-o", objFile,
		tmpFile.Name())
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

	return os.ReadFile(objFile)
}

func (f *AssemblyFile) IncludeDirectives(include []string) string {
	includes := make([]string, len(include))
	for i, inc := range include {
		includes[i] = fmt.Sprintf(".include \"%s\"", inc)
	}
	return strings.Join(includes, "\n")
}
