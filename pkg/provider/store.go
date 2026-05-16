/**********************************************************************
 * Copyright (C) 2026 Red Hat, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/

package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/openkaiden/kdn/pkg/providerservice"
	"github.com/openkaiden/kdn/pkg/secret"
)

const providersFileName = "providers.json"

// store is the unexported implementation of Store.
type store struct {
	mu          sync.Mutex
	storageDir  string
	secretStore secret.Store
}

var _ Store = (*store)(nil)

// NewStore creates a Store backed by the given storage directory.
// Secret-kind provider params are stored via the shared secret.Store so they appear in `kdn secret list`.
func NewStore(storageDir string) Store {
	return newStoreWithSecretStore(storageDir, secret.NewStore(storageDir))
}

// newStoreWithSecretStore creates a Store with an injectable secret.Store, used in tests.
func newStoreWithSecretStore(storageDir string, ss secret.Store) Store {
	return &store{storageDir: storageDir, secretStore: ss}
}

// paramRecord is the JSON-serialisable representation of a single provider parameter.
type paramRecord struct {
	Name  string                            `json:"name"`
	Kind  providerservice.ProviderParamKind `json:"kind"`
	Value string                            `json:"value"`
}

// providerRecord is the JSON-serialisable metadata for a single provider.
type providerRecord struct {
	Name   string        `json:"name"`
	Type   string        `json:"type"`
	Params []paramRecord `json:"params"`
}

type providersFile struct {
	Providers []providerRecord `json:"providers"`
}

// Create stores secret param values via the secret store and persists metadata.
// The duplicate check is performed before writing to the secret store so that an
// existing entry is never overwritten when the name is already taken.
func (s *store) Create(params CreateParams) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pf, err := s.loadProvidersFile()
	if err != nil {
		return err
	}

	for _, existing := range pf.Providers {
		if existing.Name == params.Name {
			return fmt.Errorf("provider %q: %w", params.Name, ErrProviderAlreadyExists)
		}
	}

	rec := providerRecord{
		Name:   params.Name,
		Type:   params.Type,
		Params: make([]paramRecord, 0, len(params.Params)),
	}

	for _, p := range params.Params {
		pr := paramRecord{Name: p.Name, Kind: p.Kind}
		if p.Kind == providerservice.ProviderParamKindSecret {
			// Use "{providerName}/{paramName}" as the secret name for namespacing.
			secretName := fmt.Sprintf("%s/%s", params.Name, p.Name)
			if err := s.secretStore.Create(secret.CreateParams{
				Name:        secretName,
				Type:        p.SecretType,
				Description: fmt.Sprintf("%s for %s provider", p.Name, params.Name),
				Value:       p.Value,
			}); err != nil {
				return fmt.Errorf("failed to store secret param %q: %w", p.Name, err)
			}
			pr.Value = secretName
		} else {
			pr.Value = p.Value
		}
		rec.Params = append(rec.Params, pr)
	}

	pf.Providers = append(pf.Providers, rec)
	return s.saveProvidersFile(pf)
}

// List reads providers.json and returns metadata for all stored providers.
func (s *store) List() ([]ListItem, error) {
	pf, err := s.loadProvidersFile()
	if err != nil {
		return nil, err
	}

	items := make([]ListItem, 0, len(pf.Providers))
	for _, rec := range pf.Providers {
		items = append(items, s.recordToListItem(rec))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

// Get returns the metadata and secret values for the named provider.
func (s *store) Get(name string) (ListItem, map[string]string, error) {
	pf, err := s.loadProvidersFile()
	if err != nil {
		return ListItem{}, nil, err
	}

	for _, rec := range pf.Providers {
		if rec.Name != name {
			continue
		}

		item := s.recordToListItem(rec)
		secrets := make(map[string]string)
		for _, p := range rec.Params {
			if p.Kind == providerservice.ProviderParamKindSecret {
				_, val, err := s.secretStore.Get(p.Value)
				if err != nil {
					return ListItem{}, nil, fmt.Errorf("failed to get secret param %q: %w", p.Name, err)
				}
				secrets[p.Name] = val
			}
		}
		return item, secrets, nil
	}

	return ListItem{}, nil, fmt.Errorf("provider %q: %w", name, ErrProviderNotFound)
}

// Remove deletes all secret param values and removes the provider metadata.
// Secret params that are already missing from the secret store are skipped.
func (s *store) Remove(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pf, err := s.loadProvidersFile()
	if err != nil {
		return err
	}

	idx := -1
	for i, rec := range pf.Providers {
		if rec.Name == name {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("provider %q: %w", name, ErrProviderNotFound)
	}

	for _, p := range pf.Providers[idx].Params {
		if p.Kind == providerservice.ProviderParamKindSecret {
			if err := s.secretStore.Remove(p.Value); err != nil && !errors.Is(err, secret.ErrSecretNotFound) {
				return fmt.Errorf("failed to remove secret param %q: %w", p.Name, err)
			}
		}
	}

	pf.Providers = append(pf.Providers[:idx], pf.Providers[idx+1:]...)
	return s.saveProvidersFile(pf)
}

// recordToListItem converts a providerRecord to a ListItem.
// For secret-kind params, Value holds the secret reference name (not the actual secret value).
func (s *store) recordToListItem(rec providerRecord) ListItem {
	params := make([]ProviderParamEntry, 0, len(rec.Params))
	for _, p := range rec.Params {
		params = append(params, ProviderParamEntry{Name: p.Name, Kind: p.Kind, Value: p.Value})
	}
	return ListItem{Name: rec.Name, Type: rec.Type, Params: params}
}

// loadProvidersFile reads and parses providers.json, returning an empty struct when
// the file does not yet exist.
func (s *store) loadProvidersFile() (providersFile, error) {
	var pf providersFile
	data, err := os.ReadFile(filepath.Join(s.storageDir, providersFileName))
	if os.IsNotExist(err) {
		return pf, nil
	}
	if err != nil {
		return pf, fmt.Errorf("failed to read providers file: %w", err)
	}
	if err := json.Unmarshal(data, &pf); err != nil {
		return pf, fmt.Errorf("failed to parse providers file: %w", err)
	}
	return pf, nil
}

// saveProvidersFile marshals pf and writes it to disk.
func (s *store) saveProvidersFile(pf providersFile) error {
	jsonData, err := json.MarshalIndent(pf, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal providers: %w", err)
	}

	if err := os.MkdirAll(s.storageDir, 0700); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	if err := os.WriteFile(filepath.Join(s.storageDir, providersFileName), jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write providers file: %w", err)
	}

	return nil
}
