// MIT License
//
// Copyright (c) 2026 xtaci
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do do so, subject to the following conditions:
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

// Package convolutional implements binary feedforward convolutional encoding.
package convolutional

import (
	"fmt"
	"math/bits"
)

// Encoder is a stateful rate 1/n convolutional encoder. Construct it with
// NewEncoder; its zero value is not usable. It is not safe for concurrent use.
type Encoder struct {
	generators []uint64
	mask       uint64
	state      uint64
}

// NewEncoder constructs an encoder with the given constraint length and
// generator polynomials. The constraint length must be between 1 and 64.
// Each nonzero polynomial must fit within that length. Polynomial bit 0
// selects the current input bit; bit i selects the input delayed by i steps.
// The number and order of polynomials determine the number and order of output
// bits per input bit.
func NewEncoder(constraintLength int, generators []uint64) (*Encoder, error) {
	if constraintLength < 1 || constraintLength > 64 {
		return nil, fmt.Errorf("convolutional: constraint length %d is outside [1, 64]", constraintLength)
	}
	if len(generators) == 0 {
		return nil, fmt.Errorf("convolutional: at least one generator is required")
	}
	mask := ^uint64(0) >> (64 - constraintLength)
	for i, g := range generators {
		if g == 0 || g&^mask != 0 {
			return nil, fmt.Errorf("convolutional: generator %d (%#x) is zero or exceeds constraint length %d", i, g, constraintLength)
		}
	}
	return &Encoder{
		generators: append([]uint64(nil), generators...),
		mask:       mask,
	}, nil
}

// Encode encodes bits represented as bytes containing 0 or 1. The result
// contains one byte per output bit, interleaved in generator order for each
// input bit. State is retained between calls; no termination bits are added.
// Invalid input returns an error without changing the encoder's state.
func (e *Encoder) Encode(input []byte) ([]byte, error) {
	for i, bit := range input {
		if bit > 1 {
			return nil, fmt.Errorf("convolutional: input bit %d has invalid value %d", i, bit)
		}
	}
	if len(input) > int(^uint(0)>>1)/len(e.generators) {
		return nil, fmt.Errorf("convolutional: encoded output is too large")
	}
	output := make([]byte, len(input)*len(e.generators))
	pos := 0
	for _, bit := range input {
		e.state = ((e.state << 1) | uint64(bit)) & e.mask
		for _, g := range e.generators {
			output[pos] = byte(bits.OnesCount64(e.state&g) & 1)
			pos++
		}
	}
	return output, nil
}

// Reset clears the shift register, starting a new independent stream.
func (e *Encoder) Reset() {
	e.state = 0
}
