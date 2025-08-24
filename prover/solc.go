package prover

import (
	"encoding/hex"
	"os"
	"os/exec"
	"strings"

	"github.com/erigontech/erigon-lib/crypto"
)

func CompileSolidity(source, contractName string) ([]byte, error) {
	tmpFile, err := os.CreateTemp("", "*.sol")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString(source)
	tmpFile.Close()

	cmd := exec.Command("solc", "--optimize", "--via-ir", "--bin-runtime", tmpFile.Name())
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if strings.Contains(line, contractName) && i+2 < len(lines) {
			bytecode, err := hex.DecodeString(lines[i+2])
			return bytecode, err
		}
	}

	return nil, nil
}

func EncodeCallData(functionName string) []byte {
	signature := functionName + "()"
	hash := crypto.Keccak256([]byte(signature))
	return hash[:4]
}
