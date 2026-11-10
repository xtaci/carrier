//go:build !purego && !noasm && !race

#include "textflag.h"

TEXT ·mulBlocks(SB), NOSPLIT, $0-33
	MOVD dst+0(FP), R0
	MOVD src+8(FP), R1
	MOVD n+16(FP), R2
	MOVD table+24(FP), R3
	MOVBU xor+32(FP), R4
	VLD1 (R3), [V0.B16, V1.B16]
	MOVD $15, R5
	VMOV R5, V2.B[0]
	VDUP V2.B[0], V2.B16
loop:
	VLD1.P 16(R1), [V3.B16]
	VUSHR $4, V3.B16, V4.B16
	VAND V2.B16, V3.B16, V3.B16
	VTBL V3.B16, [V0.B16], V5.B16
	VTBL V4.B16, [V1.B16], V6.B16
	VEOR V6.B16, V5.B16, V5.B16
	CBZ R4, store
	VLD1 (R0), [V7.B16]
	VEOR V7.B16, V5.B16, V5.B16
store:
	VST1.P [V5.B16], 16(R0)
	SUBS $16, R2
	BNE loop
	RET
