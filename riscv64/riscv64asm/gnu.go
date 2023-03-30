// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package riscv64asm

import (
	"strings"
)

// GNUSyntax returns the GNU assembler syntax for the instruction, as defined by GNU binutils.
// This form typically matches the syntax defined in the RISC-V Instruction Set Manual. See
// https://github.com/riscv/riscv-isa-manual/releases/download/Ratified-IMAFDQC/riscv-spec-20191213.pdf
func GNUSyntax(inst Inst) string {
	op := strings.ToLower(inst.Op.String())
	var args []string
	for _, a := range inst.Args {
		if a == nil {
			break
		}
		args = append(args, strings.ToLower(a.String()))
	}

	// No need add "I" to opcode suffix.
	// Binutils version 2.40
	var immOpcodes = map[Op]Op{
		ADDI:   ADD,
		ADDIW:  ADDW,
		ANDI:   AND,
		CSRRCI: CSRRC,
		CSRRSI: CSRRS,
		CSRRWI: CSRRW,
		ORI:    OR,
		SLLI:   SLL,
		SLLIW:  SLLW,
		SRAI:   SRA,
		SRAIW:  SRAW,
		SRLI:   SRL,
		SRLIW:  SRLW,
		XORI:   XOR,
	}

	switch inst.Op {
	case ADDI, ADDIW, ANDI, ORI, SLLI, SLLIW, SRAI, SRAIW, SRLI, SRLIW, XORI:
		op = immOpcodes[inst.Op].String()
		if inst.Op == ADDI && inst.Args[2].(Simm).Imm == 0 {
			if inst.Args[0].(Reg) == X0 && inst.Args[1].(Reg) == X0 {
				op = "nop"
				args = nil
			} else {
				op = "mv"
				args = args[:len(args)-1]
			}
		}

		if inst.Op == ADDIW && inst.Args[2].(Simm).Imm == 0 {
			op = "sext.w"
			args = args[:len(args)-1]
		}

		if inst.Op == XORI && inst.Args[2].(Simm).String() == "-1" {
			op = "not"
			args = args[:len(args)-1]
		}

	case BEQ:
		if inst.Args[1].(Reg) == X0 {
			op = "beqz"
			args[1] = args[2]
			args = args[:len(args)-1]
		}

	case BGE:
		if inst.Args[1].(Reg) == X0 {
			op = "bgez"
			args[1] = args[2]
			args = args[:len(args)-1]
		} else if inst.Args[0].(Reg) == X0 {
			op = "blez"
			args[1] = args[2]
			args = args[:len(args)-1]
		}

	case BLT:
		if inst.Args[1].(Reg) == X0 {
			op = "bltz"
			args[1] = args[2]
			args = args[:len(args)-1]
		} else if inst.Args[0].(Reg) == X0 {
			op = "bgtz"
			args[0], args[1] = args[1], args[2]
			args = args[:len(args)-1]
		}

	case BNE:
		if inst.Args[1].(Reg) == X0 {
			op = "bnez"
			args[1] = args[2]
			args = args[:len(args)-1]
		}

	case CSRRC:
		if inst.Args[0].(Reg) == X0 {
			op = "csrc"
			args[0], args[1] = args[1], args[2]
			args = args[:len(args)-1]
		}

	case CSRRCI:
		if inst.Args[0].(Reg) == X0 {
			op = "csrc"
			args[0], args[1] = args[1], args[2]
			args = args[:len(args)-1]
		} else {
			op = immOpcodes[inst.Op].String()
		}

	case CSRRS:
		if inst.Args[2].(Reg) == X0 {
			switch inst.Args[1].(Csr) {
			case FCSR:
				op = "frcsr"
				args = args[:len(args)-2]

			case FFLAGS:
				op = "frflags"
				args = args[:len(args)-2]

			case FRM:
				op = "frrm"
				args = args[:len(args)-2]

			// rdcycleh, rdinstreth and rdtimeh are RV-32 only instructions.
			// So not included there.
			case CYCLE:
				op = "rdcycle"
				args = args[:len(args)-2]

			case INSTRET:
				op = "rdinstret"
				args = args[:len(args)-2]

			case TIME:
				op = "rdtime"
				args = args[:len(args)-2]

			default:
				op = "csrr"
				args = args[:len(args)-1]
			}
		} else if inst.Args[0].(Reg) == X0 {
			op = "csrs"
			args[0], args[1] = args[1], args[2]
			args = args[:len(args)-1]
		}

	case CSRRSI:
		if inst.Args[0].(Reg) == X0 {
			op = "csrs"
			args[0], args[1] = args[1], args[2]
			args = args[:len(args)-1]
		} else {
			op = immOpcodes[inst.Op].String()
		}

	case CSRRW:
		switch inst.Args[1].(Csr) {
		case FCSR:
			op = "fscsr"
			args[1] = args[2]
			args = args[:len(args)-1]

		case FFLAGS:
			op = "fsflags"
			args[1] = args[2]
			args = args[:len(args)-1]

		case FRM:
			op = "fsrm"
			args[1] = args[2]
			args = args[:len(args)-1]

		default:
			if inst.Args[0].(Reg) == X0 {
				op = "csrw"
				args[0], args[1] = args[1], args[2]
				args = args[:len(args)-1]
			}
		}

	case CSRRWI:
		if inst.Args[0].(Reg) == X0 {
			op = "csrw"
			args[0], args[1] = args[1], args[2]
			args = args[:len(args)-1]
		} else {
			op = immOpcodes[inst.Op].String()
		}

	// When both pred and succ equals to iorw, the GNU objdump will omit them.
	case FENCE:
		if inst.Args[0].(MemOrder).String() == "iorw" &&
			inst.Args[1].(MemOrder).String() == "iorw" {
			args = nil
		}

	case FSGNJX_D:
		if inst.Args[1].(Reg) == inst.Args[2].(Reg) {
			op = "fabs.d"
			args = args[:len(args)-1]
		}

	case FSGNJX_S:
		if inst.Args[1].(Reg) == inst.Args[2].(Reg) {
			op = "fabs.s"
			args = args[:len(args)-1]
		}

	case FSGNJ_D:
		if inst.Args[1].(Reg) == inst.Args[2].(Reg) {
			op = "fmv.d"
			args = args[:len(args)-1]
		}

	case FSGNJ_S:
		if inst.Args[1].(Reg) == inst.Args[2].(Reg) {
			op = "fmv.s"
			args = args[:len(args)-1]
		}

	case FSGNJN_D:
		if inst.Args[1].(Reg) == inst.Args[2].(Reg) {
			op = "fneg.d"
			args = args[:len(args)-1]
		}

	case FSGNJN_S:
		if inst.Args[1].(Reg) == inst.Args[2].(Reg) {
			op = "fneg.s"
			args = args[:len(args)-1]
		}

	case JAL:
		if inst.Args[0].(Reg) == X0 {
			op = "j"
			args[0] = args[1]
			args = args[:len(args)-1]
		} else if inst.Args[0].(Reg) == X1 {
			op = "jal"
			args[0] = args[1]
			args = args[:len(args)-1]
		}

	case JALR:
		if inst.Args[0].(Reg) == X0 && inst.Args[1].(RegOffset).ofs.Imm == 0 {
			if inst.Args[1].(RegOffset).reg == X1 {
				op = "ret"
				args = nil
			} else {
				op = "jr"
				args[0] = inst.Args[1].(RegOffset).reg.String()
				args = args[:len(args)-1]
			}
		}

	case SLTIU:
		if inst.Args[2].(Simm).String() == "1" {
			op = "seqz"
			args = args[:len(args)-1]
		}

	case SLT:
		if inst.Args[1].(Reg) == X0 {
			op = "sgtz"
			args[1] = args[2]
			args = args[:len(args)-1]
		} else if inst.Args[2].(Reg) == X0 {
			op = "sltz"
			args = args[:len(args)-1]
		}

	case SLTU:
		if inst.Args[1].(Reg) == X0 {
			op = "snez"
			args[1] = args[2]
			args = args[:len(args)-1]
		}

	case SUB:
		if inst.Args[1].(Reg) == X0 {
			op = "neg"
			args[1] = args[2]
			args = args[:len(args)-1]
		}

	case SUBW:
		if inst.Args[1].(Reg) == X0 {
			op = "negw"
			args[1] = args[2]
			args = args[:len(args)-1]
		}
	}

	if args != nil {
		op = strings.ToLower(op)
		op += " " + strings.Join(args, ",")
	}
	return op
}
