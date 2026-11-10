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

package gf256

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
)

func referenceMul(a, b byte, polynomial uint16) byte {
	var product uint16
	for i := 0; i < 8; i++ {
		if b&(1<<i) != 0 {
			product ^= uint16(a) << i
		}
	}
	for i := 14; i >= 8; i-- {
		if product&(1<<i) != 0 {
			product ^= polynomial << (i - 8)
		}
	}
	return byte(product)
}

func mustField(t testing.TB, p uint16) *Field {
	t.Helper()
	f, err := NewField(p)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestField(t *testing.T) {
	for _, p := range []uint16{ReedSolomonPolynomial, AESPolynomial, 0x12b, 0x1f9} {
		t.Run(fmt.Sprintf("%x", p), func(t *testing.T) {
			f := mustField(t, p)
			if f.Polynomial() != p {
				t.Fatalf("Polynomial() = %#x", f.Polynomial())
			}
			for a := 0; a < 256; a++ {
				for b := 0; b < 256; b++ {
					got := f.Mul(byte(a), byte(b))
					want := referenceMul(byte(a), byte(b), p)
					if got != want {
						t.Fatalf("%x * %x = %x, want %x", a, b, got, want)
					}
					if Add(byte(a), byte(b)) != byte(a^b) || Sub(byte(a), byte(b)) != byte(a^b) {
						t.Fatal("addition or subtraction mismatch")
					}
					if b != 0 && f.Mul(f.Div(byte(a), byte(b)), byte(b)) != byte(a) {
						t.Fatalf("division failed: %x / %x", a, b)
					}
				}
				if a != 0 && f.Mul(byte(a), f.Inv(byte(a))) != 1 {
					t.Fatalf("inverse failed: %x", a)
				}
				if f.Pow(byte(a), 0) != 1 || f.Pow(byte(a), 1) != byte(a) {
					t.Fatalf("power identity failed: %x", a)
				}
				if a != 0 && f.Pow(byte(a), 255) != 1 {
					t.Fatalf("Fermat identity failed: %x", a)
				}
			}
		})
	}
	if got := mustField(t, AESPolynomial).Mul(0x57, 0x83); got != 0xc1 {
		t.Fatalf("AES example: got %#x, want 0xc1", got)
	}
}

func TestNewFieldValidation(t *testing.T) {
	for _, p := range []uint16{0, 0xff, 0x100, 0x101, 0x11a, 0x200, 0xffff} {
		if _, err := NewField(p); err == nil {
			t.Errorf("NewField(%#x) accepted invalid polynomial", p)
		}
	}
	// A degree-eight quotient is a field exactly when it has no zero divisors.
	count := 0
	for p := uint16(0x100); p < 0x200; p++ {
		field := true
		for a := 1; a < 256 && field; a++ {
			for b := 1; b < 256; b++ {
				if referenceMul(byte(a), byte(b), p) == 0 {
					field = false
					break
				}
			}
		}
		f, err := NewField(p)
		if (err == nil) != field {
			t.Fatalf("NewField(%#x) field=%v, error=%v", p, field, err)
		}
		if field {
			count++
			for a := 1; a < 256; a++ {
				if referenceMul(byte(a), f.Inv(byte(a)), p) != 1 {
					t.Fatalf("incorrect inverse for polynomial %#x, element %#x", p, a)
				}
			}
		}
	}
	if count != 30 {
		t.Fatalf("found %d irreducible polynomials, want 30", count)
	}
}

func TestUnalignedSlices(t *testing.T) {
	f := mustField(t, ReedSolomonPolynomial)
	for srcOffset := 0; srcOffset < 16; srcOffset++ {
		for dstOffset := 0; dstOffset < 16; dstOffset++ {
			for _, xor := range []bool{false, true} {
				srcBuffer := bytes.Repeat([]byte{0xa5}, 100)
				dstBuffer := bytes.Repeat([]byte{0x5a}, 100)
				src := srcBuffer[srcOffset : srcOffset+65]
				dst := dstBuffer[dstOffset : dstOffset+65]
				for i := range src {
					src[i] = byte(i * 7)
				}
				want := bytes.Clone(dstBuffer)
				for i, value := range src {
					product := referenceMul(value, 73, ReedSolomonPolynomial)
					if xor {
						want[dstOffset+i] ^= product
					} else {
						want[dstOffset+i] = product
					}
				}
				if xor {
					f.MulSliceXor(dst, src, 73)
				} else {
					f.MulSlice(dst, src, 73)
				}
				if !bytes.Equal(dstBuffer, want) {
					t.Fatalf("src offset=%d dst offset=%d xor=%v", srcOffset, dstOffset, xor)
				}
			}
		}
	}
}

func expectPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Error("expected panic")
		}
	}()
	fn()
}

func TestZeroDivision(t *testing.T) {
	f := mustField(t, ReedSolomonPolynomial)
	expectPanic(t, func() { f.Inv(0) })
	expectPanic(t, func() { f.Div(0, 0) })
	expectPanic(t, func() { f.Div(1, 0) })
}

func TestSlices(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for _, p := range []uint16{ReedSolomonPolynomial, AESPolynomial} {
		f := mustField(t, p)
		for _, size := range []int{0, 1, 15, 16, 17, 31, 32, 33, 63, 64, 65, 255, 1024} {
			for c := 0; c < 256; c++ {
				src := make([]byte, size)
				initial := make([]byte, size+3)
				rng.Read(src)
				rng.Read(initial)
				for _, xor := range []bool{false, true} {
					dst := bytes.Clone(initial)
					want := bytes.Clone(initial)
					for i, b := range src {
						product := referenceMul(byte(c), b, p)
						if xor {
							want[i] ^= product
						} else {
							want[i] = product
						}
					}
					if xor {
						f.MulSliceXor(dst, src, byte(c))
					} else {
						f.MulSlice(dst, src, byte(c))
					}
					if !bytes.Equal(dst, want) {
						t.Fatalf("p=%#x size=%d c=%d xor=%v: got %x want %x", p, size, c, xor, dst, want)
					}

					inplace := bytes.Clone(src)
					for i, b := range src {
						want[i] = referenceMul(byte(c), b, p)
						if xor {
							want[i] ^= b
						}
					}
					if xor {
						f.MulSliceXor(inplace, inplace, byte(c))
					} else {
						f.MulSlice(inplace, inplace, byte(c))
					}
					if !bytes.Equal(inplace, want[:size]) {
						t.Fatalf("in-place p=%#x size=%d c=%d xor=%v", p, size, c, xor)
					}
				}
			}
		}
	}
}

func TestSliceValidation(t *testing.T) {
	f := mustField(t, ReedSolomonPolynomial)
	for _, xor := range []bool{false, true} {
		for _, c := range []byte{0, 1, 2} {
			call := func(dst, src []byte) {
				if xor {
					f.MulSliceXor(dst, src, c)
				} else {
					f.MulSlice(dst, src, c)
				}
			}
			expectPanic(t, func() { call(make([]byte, 3), make([]byte, 4)) })
			for _, reversed := range []bool{false, true} {
				buf := []byte{1, 2, 3, 4, 5, 6}
				before := bytes.Clone(buf)
				a, b := buf[:4], buf[1:5]
				if reversed {
					a, b = b, a
				}
				expectPanic(t, func() { call(a, b) })
				if !bytes.Equal(buf, before) {
					t.Fatal("invalid input modified buffer")
				}
			}
			buf := []byte{1, 2, 3, 4}
			call(buf[2:], buf[:2])
		}
	}
}

func BenchmarkMulSlice(b *testing.B) {
	f := mustField(b, ReedSolomonPolynomial)
	for _, size := range []int{64, 1024, 64 * 1024} {
		for _, xor := range []bool{false, true} {
			b.Run(fmt.Sprintf("size=%d/xor=%t", size, xor), func(b *testing.B) {
				src, dst := make([]byte, size), make([]byte, size)
				rand.New(rand.NewSource(1)).Read(src)
				b.SetBytes(int64(size))
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if xor {
						f.MulSliceXor(dst, src, 37)
					} else {
						f.MulSlice(dst, src, 37)
					}
				}
			})
		}
	}
}
