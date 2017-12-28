// Reference: www.ssrc.ucsc.edu/Papers/plank-fast13.pdf

#include "textflag.h"

// func gfMulAVX2(low, high, in, out []byte)
TEXT ·gfMulAVX2(SB), NOSPLIT, $0
	// table -> ymm
	MOVQ    lowTable+0(FP), AX   // it's not just MOVQ, it's more like MOV
	MOVQ    highTable+24(FP), BX
	VMOVDQU (AX), X0             // 128-bit Intel® AVX instructions operate on the lower 128 bits of the YMM registers and zero the upper 128 bits
	VMOVDQU (BX), X1             // avoiding AVX-SSE Transition Penalties
	// [0..0,X0] -> [X0, X0]
	VINSERTI128 $1, X0, Y0, Y0 // low_table -> ymm0
	VINSERTI128 $1, X1, Y1, Y1 // high_table -> ymm1

	MOVQ        in+48(FP), AX  // in_addr -> AX
	MOVQ        out+72(FP), BX // out_addr -> BX

	// mask -> ymm
	WORD $0x0fb2                    // MOV $0x0f, DL. Please don't use R8-R15 here, because it need one more byte for instruction decode
	LONG $0x2069e3c4; WORD $0x00d2  // VPINSRB $0x00, EDX, XMM2, XMM2
	VPBROADCASTB X2, Y2             // [1111,1111,1111...1111]

	// if done
	MOVQ  in_len+56(FP), CX // in_len -> CX
	SHRQ  $5, CX            // CX = CX >> 5 (calc 32bytes per loop)
	TESTQ CX, CX            // bitwise AND on two operands,if result is 0 (it means no more data)，ZF flag set 1
	JZ    done              // jump to done if ZF is 0

loop:
	// split data byte into two 4-bit
	VMOVDQU (AX), Y4   // in_data -> ymm4
	VPSRLQ  $4, Y4, Y5 // shift in_data's 4 high bit to low -> ymm5
	VPAND   Y2, Y5, Y5 // mask AND data_shift -> ymm5 (high data)
	VPAND   Y2, Y4, Y4 // mask AND data -> ymm4 (low data)

	// shuffle table
	VPSHUFB Y5, Y1, Y6
	VPSHUFB Y4, Y0, Y7

	// gf add low, high 4-bit & output
	VPXOR   Y6, Y7, Y3
	VMOVDQU Y3, (BX)   // it will loss performance if use Non-Temporal Hint here, because "out" will be read for next data shard encoding

	// next loop
	ADDQ $32, AX
	ADDQ $32, BX
	SUBQ $1, CX  // it will affect ZF
	JNZ  loop

done:
	RET

// almost same with gfMulAVX2
// two more steps: 1. get the old out_data 2. update the out_data
// func gfMulXorAVX2(low, high, in, out []byte)
TEXT ·gfMulXorAVX2(SB), NOSPLIT, $0
	MOVQ         lowTable+0(FP), AX
	MOVQ         highTable+24(FP), BX
	VMOVDQU      (AX), X0
	VMOVDQU      (BX), X1
	VINSERTI128  $1, X0, Y0, Y0
	VINSERTI128  $1, X1, Y1, Y1
	MOVQ         in+48(FP), AX
	MOVQ         out+72(FP), BX
	WORD $0x0fb2
	LONG $0x2069e3c4; WORD $0x00d2
	VPBROADCASTB X2, Y2
	MOVQ         in_len+56(FP), CX
	SHRQ         $5, CX
	TESTQ        CX, CX
	JZ           done

loop:
	VMOVDQU (AX), Y4
	VMOVDQU (BX), Y3   // out_data -> Ymm
	VPSRLQ  $4, Y4, Y5
	VPAND   Y2, Y5, Y5
	VPAND   Y2, Y4, Y4
	VPSHUFB Y5, Y1, Y6
	VPSHUFB Y4, Y0, Y7
	VPXOR   Y6, Y7, Y6
	VPXOR   Y6, Y3, Y3 // update result
	VMOVDQU Y3, (BX)
	ADDQ    $32, AX
	ADDQ    $32, BX
	SUBQ    $1, CX
	JNZ     loop

done:
	RET

// func hasAVX2() bool
TEXT ·hasAVX2(SB), NOSPLIT, $0
	CMPB runtime·support_avx2(SB), $1
	JE   has
	MOVB $0, ret+0(FP)
	RET

has:
	MOVB $1, ret+0(FP)
	RET



