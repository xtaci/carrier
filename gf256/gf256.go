// MIT License
//
// Copyright (c) 2026 xtaci
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// Package gf256 implements arithmetic in binary extension fields GF(2^8).
//
// Fields use the polynomial basis: bit i represents the coefficient of x^i.
// Field values are immutable after construction and safe for concurrent use.
// The table-based operations are not constant-time and must not be used where
// operands are secret.
package gf256

import (
	"fmt"
	"unsafe"
)

const (
	// ReedSolomonPolynomial is x^8+x^4+x^3+x^2+1.
	ReedSolomonPolynomial uint16 = 0x11d
	// AESPolynomial is x^8+x^4+x^3+x+1.
	AESPolynomial uint16 = 0x11b
)

// Field is a GF(2^8) field. Construct it with NewField; its zero value is
// not usable.
type Field struct {
	polynomial uint16
	mul        [256][256]byte
	inverse    [256]byte
	nibbles    [256][32]byte
}

// NewField constructs a field for a degree-eight irreducible polynomial,
// including its x^8 bit. Reducible or out-of-range polynomials are rejected.
// Construction allocates and initializes lookup tables; reuse the result.
func NewField(polynomial uint16) (*Field, error) {
	if polynomial < 0x100 || polynomial > 0x1ff || !irreducible(polynomial) {
		return nil, fmt.Errorf("gf256: polynomial %#x is not irreducible of degree 8", polynomial)
	}
	f := &Field{polynomial: polynomial}
	for a := 0; a < 256; a++ {
		for b := 0; b < 256; b++ {
			f.mul[a][b] = multiply(byte(a), byte(b), byte(polynomial))
			if f.mul[a][b] == 1 {
				f.inverse[a] = byte(b)
			}
		}
		for i := 0; i < 16; i++ {
			f.nibbles[a][i] = f.mul[a][i]
			f.nibbles[a][16+i] = f.mul[a][i<<4]
		}
	}
	return f, nil
}

func irreducible(p uint16) bool {
	// A reducible degree-eight polynomial has a factor of degree at most four.
	for degree := 1; degree <= 4; degree++ {
		for divisor := uint16(1 << degree); divisor < uint16(1<<(degree+1)); divisor++ {
			r := p
			for bit := 8; bit >= degree; bit-- {
				if r&(1<<bit) != 0 {
					r ^= divisor << (bit - degree)
				}
			}
			if r == 0 {
				return false
			}
		}
	}
	return true
}

func multiply(a, b, reduction byte) byte {
	var result byte
	for b != 0 {
		if b&1 != 0 {
			result ^= a
		}
		high := a & 0x80
		a <<= 1
		if high != 0 {
			a ^= reduction
		}
		b >>= 1
	}
	return result
}

// Polynomial returns the field's defining polynomial, including the x^8 bit.
func (f *Field) Polynomial() uint16 { return f.polynomial }

// Add and Sub are addition and subtraction in characteristic two.
func Add(a, b byte) byte { return a ^ b }

// Sub subtracts b from a.
func Sub(a, b byte) byte { return a ^ b }

// Mul returns the product of a and b.
func (f *Field) Mul(a, b byte) byte { return f.mul[a][b] }

// Inv returns the multiplicative inverse of a. It panics if a is zero.
func (f *Field) Inv(a byte) byte {
	if a == 0 {
		panic("gf256: inverse of zero")
	}
	return f.inverse[a]
}

// Div divides a by b. It panics if b is zero.
func (f *Field) Div(a, b byte) byte {
	return f.mul[a][f.Inv(b)]
}

// Pow returns a raised to n. In particular, Pow(0, 0) is 1.
func (f *Field) Pow(a byte, n uint64) byte {
	result := byte(1)
	for n != 0 {
		if n&1 != 0 {
			result = f.mul[result][a]
		}
		a = f.mul[a][a]
		n >>= 1
	}
	return result
}

// MulSlice sets dst[i] = c * src[i] for each element of src.
// dst must be at least as long as src. Exact overlap is allowed, but inexact
// overlap panics. Bytes beyond len(src) in dst are not modified.
func (f *Field) MulSlice(dst, src []byte, c byte) {
	dst = checkedSlices(dst, src)
	switch c {
	case 0:
		clear(dst)
		return
	case 1:
		copy(dst, src)
		return
	}
	n := mulSliceSIMD(dst, src, &f.nibbles[c], false)
	table := &f.mul[c]
	for i := n; i < len(src); i++ {
		dst[i] = table[src[i]]
	}
}

// MulSliceXor sets dst[i] ^= c * src[i] for each element of src.
// It has the same length and overlap requirements as MulSlice.
func (f *Field) MulSliceXor(dst, src []byte, c byte) {
	dst = checkedSlices(dst, src)
	if c == 0 {
		return
	}
	n := mulSliceSIMD(dst, src, &f.nibbles[c], true)
	table := &f.mul[c]
	for i := n; i < len(src); i++ {
		dst[i] ^= table[src[i]]
	}
}

func checkedSlices(dst, src []byte) []byte {
	if len(dst) < len(src) {
		panic("gf256: destination shorter than source")
	}
	dst = dst[:len(src)]
	if len(src) == 0 {
		return dst
	}
	d := uintptr(unsafe.Pointer(unsafe.SliceData(dst)))
	s := uintptr(unsafe.Pointer(unsafe.SliceData(src)))
	if d != s && d < s+uintptr(len(src)) && s < d+uintptr(len(dst)) {
		panic("gf256: invalid buffer overlap")
	}
	return dst
}
