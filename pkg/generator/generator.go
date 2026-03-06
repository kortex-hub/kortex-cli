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
	"encoding/hex"
	"io"
	"math/big"
	mathrand "math/rand"
	"sync"
	"time"
)

// Generator generates unique identifiers
type Generator interface {
	// Generate creates a unique identifier
	Generate() string
}

// generator is the internal implementation of Generator
type generator struct {
	reader io.Reader
}

// Compile-time check to ensure generator implements Generator interface
var _ Generator = (*generator)(nil)

// New creates a new ID generator with the default random reader
func New() Generator {
	return newWithReader(newMathRandReader())
}

// newWithReader creates a new ID generator with a custom reader.
// This is unexported and primarily useful for testing with fake readers.
func newWithReader(reader io.Reader) Generator {
	return &generator{reader: reader}
}

// Generate creates a unique identifier.
// Uses math/rand for random generation.
func (g *generator) Generate() string {
	b := make([]byte, 32)
	for {
		if _, err := io.ReadFull(g.reader, b); err != nil {
			panic(err) // This shouldn't happen
		}
		id := hex.EncodeToString(b)
		// Avoid all-numeric values to prevent auto conversion issues
		if _, ok := new(big.Int).SetString(id, 10); ok {
			continue
		}
		return id
	}
}

// newMathRandReader creates a new math/rand-based reader
func newMathRandReader() io.Reader {
	return &mathRandReader{rand: mathrand.New(mathrand.NewSource(time.Now().UnixNano()))}
}

// mathRandReader wraps math/rand.Rand to implement io.Reader
// with thread-safe access
type mathRandReader struct {
	rand *mathrand.Rand
	mu   sync.Mutex
}

func (r *mathRandReader) Read(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.rand.Read(p)
}
