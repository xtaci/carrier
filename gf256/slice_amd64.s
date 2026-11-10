//go:build !purego && !noasm && !race

#include "textflag.h"

TEXT ·hasSSSE3(SB), NOSPLIT, $0-1
	MOVL $1, AX
	CPUID
	SHRL $9, CX
	ANDL $1, CX
	MOVB CX, ret+0(FP)
	RET

// The two shuffle tables map the low and high nibbles to partial products.
TEXT ·mulBlocks(SB), NOSPLIT, $0-33
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ n+16(FP), CX
	MOVQ table+24(FP), AX
	MOVBLZX xor+32(FP), DX
	MOVOU (AX), X0
	MOVOU 16(AX), X1
	MOVQ $0x0f0f0f0f0f0f0f0f, AX
	MOVQ AX, X2
	PUNPCKLQDQ X2, X2
loop:
	MOVOU (SI), X3
	MOVOU X3, X4
	PAND X2, X3
	PSRLW $4, X4
	PAND X2, X4
	MOVOU X0, X5
	MOVOU X1, X6
	PSHUFB X3, X5
	PSHUFB X4, X6
	PXOR X6, X5
	TESTL DX, DX
	JZ store
	MOVOU (DI), X7
	PXOR X7, X5
store:
	MOVOU X5, (DI)
	ADDQ $16, SI
	ADDQ $16, DI
	SUBQ $16, CX
	JNZ loop
	RET
