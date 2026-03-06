// Copyright 2026 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package generator

import (
	mathrand "math/rand"
	"testing"
)

func TestGenerate(t *testing.T) {
	t.Parallel()

	gen := New()

	t.Run("generates non-empty ID", func(t *testing.T) {
		t.Parallel()

		id := gen.Generate()
		if id == "" {
			t.Error("Generate() returned empty ID")
		}
	})

	t.Run("generates ID with correct length", func(t *testing.T) {
		t.Parallel()

		id := gen.Generate()
		// 32 bytes * 2 hex chars per byte = 64 characters
		expectedLength := 64
		if len(id) != expectedLength {
			t.Errorf("Generate() returned ID with length %d, want %d", len(id), expectedLength)
		}
	})

	t.Run("skips all-numeric IDs and returns alphanumeric", func(t *testing.T) {
		t.Parallel()

		// Create a fake reader that returns all-numeric first, then alphanumeric
		reader := &sequentialReader{
			sequences: [][]byte{
				// First sequence: all zeros -> encodes to all "00" (numeric)
				make([]byte, 32), // Initialized to all zeros
				// Second sequence: includes 0xaa -> encodes to "aa" (alphanumeric)
				{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x11, 0x22,
					0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa,
					0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x11, 0x22, 0x33,
					0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb},
			},
		}

		gen := newWithReader(reader)
		id := gen.Generate()

		// Should skip the all-numeric and return the alphanumeric one
		expectedID := "aabbccddeeff112233445566778899aabbccddeeff112233445566778899aabb"
		if id != expectedID {
			t.Errorf("Generate() = %v, want %v", id, expectedID)
		}

		// Verify it was called twice (once for numeric, once for alphanumeric)
		if reader.callCount != 2 {
			t.Errorf("Reader was called %d times, want 2", reader.callCount)
		}
	})
}

func TestNewWithReader(t *testing.T) {
	t.Parallel()

	t.Run("uses custom reader", func(t *testing.T) {
		t.Parallel()

		// Create a deterministic reader for testing
		// The pattern {0xaa, 0xbb, 0xcc, 0xdd} repeated 8 times = 32 bytes
		reader := &fakeReader{data: []byte{0xaa, 0xbb, 0xcc, 0xdd}}
		gen := newWithReader(reader)

		id := gen.Generate()

		// Verify the ID matches the expected hex encoding
		// 0xaabbccdd repeated 8 times = 32 bytes hex encoded to 64 chars
		expectedID := "aabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccdd"
		if id != expectedID {
			t.Errorf("Generate() with custom reader = %v, want %v", id, expectedID)
		}
	})
}

func TestMathRandReader(t *testing.T) {
	t.Parallel()

	t.Run("reads requested number of bytes", func(t *testing.T) {
		t.Parallel()

		reader := &mathRandReader{rand: mathrand.New(mathrand.NewSource(12345))}
		buf := make([]byte, 32)

		n, err := reader.Read(buf)
		if err != nil {
			t.Fatalf("Read() unexpected error = %v", err)
		}
		if n != 32 {
			t.Errorf("Read() returned n = %d, want 32", n)
		}
	})
}

// fakeReader is a test helper that implements io.Reader with deterministic data
type fakeReader struct {
	data []byte
	pos  int
}

func (r *fakeReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		r.pos = 0 // Loop back to start
	}

	n = copy(p, r.data[r.pos:])
	if n < len(p) {
		// Need to wrap around or repeat
		for n < len(p) {
			copied := copy(p[n:], r.data)
			n += copied
			if copied == 0 {
				break
			}
		}
	}

	r.pos += n
	return n, nil
}

// sequentialReader returns different byte sequences on each call
type sequentialReader struct {
	sequences [][]byte
	callCount int
}

func (r *sequentialReader) Read(p []byte) (n int, err error) {
	if r.callCount >= len(r.sequences) {
		// Return the last sequence for any additional calls
		r.callCount++
		return copy(p, r.sequences[len(r.sequences)-1]), nil
	}

	n = copy(p, r.sequences[r.callCount])
	r.callCount++
	return n, nil
}
