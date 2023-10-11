package bootstrap

import (
	"encoding/base64"

	"github.com/open-component-model/mpas/internal/env"
)

// SetProviderDataFormat takes some data and updates it according to the provider's need.
// If the provider is not known, it returns the data as is unchanged, but converted to string.
func SetProviderDataFormat(provider string, data []byte) string {
	switch provider {
	case env.ProviderGitea:
		return base64.StdEncoding.EncodeToString(data)
	}

	return string(data)
}
