// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentsgen

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Binary(t *testing.T) {
	l, err := getFreePort()
	require.NoError(t, err)
	defer l.Close()
	port := l.Addr().(*net.TCPAddr).Port
	tmpDir := t.TempDir()
	text := []byte("binary")
	checksum, err := computeHash(text)
	require.NoError(t, err)
	checks := fmt.Sprintf("%s  %s", checksum, fmt.Sprintf("http://localhost:%d/%s", port, "binary"))
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/binary":
			w.WriteHeader(http.StatusOK)
			w.Write(text)
		case "/hash":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(checks))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	server.Listener.Close()
	server.Listener = l
	server.Start()
	defer server.Close()

	testCases := []struct {
		name        string
		version     string
		expectedErr bool
	}{
		{
			name:    "valid version",
			version: "v1.0.0",
		},
		{
			name:        "invalid version",
			version:     "1.0.0.0",
			expectedErr: true,
		},
		{
			name:        "latest version",
			version:     "latest",
			expectedErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			b := &Binary{
				Version: tc.version,
				BinURL:  server.URL + "/binary",
				HashURL: server.URL + "/hash",
			}
			ctx := context.Background()
			err = b.Get(ctx, tmpDir)
			if tc.expectedErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, text, b.Content)
		})
	}
}

func getFreePort() (net.Listener, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, err
	}

	return l, nil
}
