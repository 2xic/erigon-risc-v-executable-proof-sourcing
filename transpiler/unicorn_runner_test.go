package transpiler

import (
	"erigon-transpiler-risc-v/prover"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func TestBigIntConverter(t *testing.T) {
	file := prover.AssemblyFile{
		Instructions: []prover.Instruction{
			{Name: "mv", Operands: []string{"s2", "sp"}},
			{Name: "addi", Operands: []string{"sp", "sp", "-80"}},

			// 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF
			{Name: "li", Operands: []string{"t0", "0xFFFFFFFF"}},
			{Name: "sw", Operands: []string{"t0", "0(sp)"}},
			{Name: "sw", Operands: []string{"t0", "4(sp)"}},
			{Name: "sw", Operands: []string{"t0", "8(sp)"}},
			{Name: "sw", Operands: []string{"t0", "12(sp)"}},
			{Name: "sw", Operands: []string{"t0", "16(sp)"}},
			{Name: "sw", Operands: []string{"t0", "20(sp)"}},
			{Name: "sw", Operands: []string{"t0", "24(sp)"}},
			{Name: "sw", Operands: []string{"t0", "28(sp)"}},
			{Name: "ebreak", Operands: []string{}},

			// 0x1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF
			{Name: "li", Operands: []string{"t0", "0x90ABCDEF"}},
			{Name: "li", Operands: []string{"t1", "0x12345678"}},
			{Name: "sw", Operands: []string{"t0", "32(sp)"}},
			{Name: "sw", Operands: []string{"t1", "36(sp)"}},
			{Name: "sw", Operands: []string{"t0", "40(sp)"}},
			{Name: "sw", Operands: []string{"t1", "44(sp)"}},
			{Name: "sw", Operands: []string{"t0", "48(sp)"}},
			{Name: "sw", Operands: []string{"t1", "52(sp)"}},
			{Name: "sw", Operands: []string{"t0", "56(sp)"}},
			{Name: "sw", Operands: []string{"t1", "60(sp)"}},
			{Name: "ebreak", Operands: []string{}},

			//  Add thw two 256-bit numbers
			{Name: "addi", Operands: []string{"a0", "sp", "0"}},
			{Name: "addi", Operands: []string{"a1", "sp", "32"}},
			{Name: "addi", Operands: []string{"a2", "sp", "32"}},
			{Name: "call", Operands: []string{"add256_stack_scratch"}},
			{Name: "ebreak", Operands: []string{}},

			{Name: "mv", Operands: []string{"sp", "s2"}},

			{Name: "jr", Operands: []string{"x0"}},
		},
	}

	bytecode, err := file.ToBytecode()
	assert.NoError(t, err)

	VmRunner, err := prover.NewUnicornRunner()
	assert.NoError(t, err)

	snapshot, err := VmRunner.Execute(bytecode)
	assert.NoError(t, err)
	assert.NotNil(t, snapshot)

	snapShot := *snapshot.StackSnapshots
	firstValue, err := uint256.FromHex("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")
	assert.NoError(t, err)

	secondValue, err := uint256.FromHex("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	assert.NoError(t, err)

	resultsValue, err := uint256.FromHex("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdee")
	assert.NoError(t, err)

	assert.Equal(t, []uint256.Int{
		*uint256.NewInt(0x0),
		*firstValue,
	}, snapShot[0])
	assert.Equal(t, []uint256.Int{
		*secondValue,
		*firstValue,
	}, snapShot[1])
	assert.Equal(t, []uint256.Int{
		*resultsValue,
	}, snapShot[2])
}
