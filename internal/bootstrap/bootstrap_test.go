// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"testing"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/stretchr/testify/assert"
)

func Test_GetOrderedKeys(t *testing.T) {
	testsCases := []struct {
		name     string
		input    map[string]compdesc.ComponentReference
		expected []string
	}{
		{
			name:     "empty map",
			input:    map[string]compdesc.ComponentReference{},
			expected: []string{},
		},
		{
			name: "map with one key",
			input: map[string]compdesc.ComponentReference{
				"key1": {},
			},
			expected: []string{"key1"},
		},
		{
			name: "map with two keys",
			input: map[string]compdesc.ComponentReference{
				"key1": {},
				"key2": {},
			},
			expected: []string{"key1", "key2"},
		},
		{
			name: "map with unordered keys",
			input: map[string]compdesc.ComponentReference{
				"key2": {},
				"key1": {},
			},
			expected: []string{"key1", "key2"},
		},
		{
			name: "map with alphabetically unordered keys",
			input: map[string]compdesc.ComponentReference{
				"a-key": {},
				"b-key": {},
				"w-key": {},
				"p-key": {},
				"z-key": {},
			},
			expected: []string{"a-key", "b-key", "p-key", "w-key", "z-key"},
		},
	}

	for _, tc := range testsCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := getOrderedKeys(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
