package prover

import (
	"bytes"
	_ "embed"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/holiman/uint256"
)

const InstructionEBREAK = "EBREAK"

type AssemblyFile struct {
	Instructions []Instruction
	DataSection  []DataVariable
}

type DataVariable struct {
	Name  string
	Value *uint256.Int
}

//go:embed resources/lib.asm
var libFile []byte

type Instruction struct {
	Name     string
	Operands []string
}

func (a *AssemblyFile) toDebugFile() string {
	instructions := a.toFile(false)
	dataSection := a.generateDataSection()
	file := `
.section .data
%s

.section .text
.global execute
execute:
%s 
    jr x0
%s
	`
	content := fmt.Sprintf(file, dataSection, instructions, string(libFile))
	return content
}

func (a *AssemblyFile) generateDataSection() string {
	if len(a.DataSection) == 0 {
		return ""
	}

	var lines []string
	for _, dataVar := range a.DataSection {
		bytes := dataVar.Value.Bytes32()
		lines = append(lines, fmt.Sprintf("%s:", dataVar.Name))
		for i := 0; i < 8; i++ {
			offset := (7 - i) * 4
			word := uint32(bytes[offset+3]) |
				uint32(bytes[offset+2])<<8 |
				uint32(bytes[offset+1])<<16 |
				uint32(bytes[offset])<<24
			lines = append(lines, fmt.Sprintf("    .word 0x%08x", word))
		}
	}
	return strings.Join(lines, "\n")
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

	fmt.Println(libFile)

	content := strings.Join(instructions, "\n")
	return content
}

func (f *AssemblyFile) ToToolChainCompatibleAssembly() (string, error) {
	dataSection := f.generateDataSection()
	var format string
	if dataSection != "" {
		format = `
.section .data
%s

.section .text
.global execute
execute:
	# Save stack
	mv s2, sp
	mv s1, ra

%s

	# Restore stack
	mv sp, s2
	mv ra, s1
	ret	
%s
	`
		return fmt.Sprintf(format, dataSection, f.toZkFile(), libFile), nil
	} else {
		format = `
.global execute
execute:
	# Save stack
	mv s2, sp
	mv s1, ra

%s

	# Restore stack
	mv sp, s2
	mv ra, s1
	ret	
%s
	`
		return fmt.Sprintf(format, f.toZkFile(), libFile), nil
	}
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
		"-nostdlib",
		"-nostartfiles",
		"-static",
		"-Wl,--entry=execute",
		"-Wl,-Ttext=0x1000",
		"-Wl,-Tdata=0x12000",
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
