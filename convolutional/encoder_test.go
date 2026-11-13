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

package convolutional

import (
	"bytes"
	"testing"
)

func TestEncode(t *testing.T) {
	generators := []uint64{0o7, 0o5}
	e, err := NewEncoder(3, generators)
	if err != nil {
		t.Fatal(err)
	}
	// Mutating the caller's slice must not change the encoder.
	generators[0] = 0
	input := []byte{1, 0, 1, 1, 0, 0}
	want := []byte{1, 1, 1, 0, 0, 0, 0, 1, 0, 1, 1, 1}
	got, err := e.Encode(input)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("Encode() = %v, %v; want %v", got, err, want)
	}

	e.Reset()
	first, err := e.Encode(input[:2])
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.Encode([]byte{1, 2}); err == nil {
		t.Fatal("invalid input accepted")
	}
	second, err := e.Encode(input[2:])
	if err != nil {
		t.Fatal(err)
	}
	if got := append(first, second...); !bytes.Equal(got, want) {
		t.Fatalf("chunked encoding = %v, want %v", got, want)
	}
	e.Reset()
	got, err = e.Encode(input)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("encoding after reset = %v, %v; want %v", got, err, want)
	}
}

func TestNewEncoderValidation(t *testing.T) {
	for _, tc := range []struct {
		length int
		gens   []uint64
	}{
		{0, []uint64{1}},
		{65, []uint64{1}},
		{3, nil},
		{3, []uint64{0}},
		{3, []uint64{8}},
	} {
		if _, err := NewEncoder(tc.length, tc.gens); err == nil {
			t.Errorf("NewEncoder(%d, %v) accepted invalid parameters", tc.length, tc.gens)
		}
	}
}

func TestDelayedAndWideGenerators(t *testing.T) {
	for _, length := range []int{1, 3, 64} {
		e, err := NewEncoder(length, []uint64{1, uint64(1) << (length - 1)})
		if err != nil {
			t.Fatal(err)
		}
		input := make([]byte, length+1)
		input[0] = 1
		got, err := e.Encode(input)
		if err != nil {
			t.Fatal(err)
		}
		want := make([]byte, len(input)*2)
		want[0] = 1
		want[2*(length-1)+1] = 1
		if !bytes.Equal(got, want) {
			t.Errorf("length %d: got %v, want %v", length, got, want)
		}
	}
}
