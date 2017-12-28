// Reference: www.ssrc.ucsc.edu/Papers/plank-fast13.pdf

#include "textflag.h"

// func xorAVX2(in, out []byte)
TEXT ·xorAVX2(SB), NOSPLIT, $0
	MOVQ  in+0(FP), AX
	MOVQ  out+24(FP), BX
	MOVQ  in_len+8(FP), CX
	SHRQ  $5, CX
	TESTQ CX, CX
	JZ    done

loop:
	VMOVDQU (AX), Y4
	VMOVDQU (BX), Y3   // out_data -> Ymm
	VPXOR   Y4, Y3, Y4
	VMOVDQU Y4, (BX)
	ADDQ    $32, AX
	ADDQ    $32, BX
	SUBQ    $1, CX
	JNZ     loop

done:
	RET

// func copyAVX2(in, out []byte)
TEXT ·copyAVX2(SB), NOSPLIT, $0
	MOVQ  in+0(FP), AX
	MOVQ  out+24(FP), BX
	MOVQ  in_len+8(FP), CX
	SHRQ  $5, CX
	TESTQ CX, CX
	JZ    done

loop:
	VMOVDQU (AX), Y4
	VMOVDQU Y4, (BX)
	ADDQ    $32, AX
	ADDQ    $32, BX
	SUBQ    $1, CX
	JNZ     loop

done:
	RET
