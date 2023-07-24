// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"os"
	"path"

	"github.com/open-component-model/mpas/cmd/mpas/config"
	"github.com/open-component-model/mpas/pkg/oci"
	"github.com/open-component-model/mpas/pkg/printer"
)

// Export exports the latest version of the component with the given name to the given path.
func Export(ctx context.Context, cfg *config.MpasConfig, repositoryURL string) error {
	repo := oci.Repository{
		RepositoryURL: repositoryURL,
	}
	ver, err := repo.GetLatestVersion(ctx)
	if err != nil {
		return err
	}

	cfg.Printer.Printf("Downloading bootstrap component %s with version %s ...\n",
		printer.BoldBlue(repositoryURL),
		printer.BoldBlue(ver))

	name, err := repo.PullArtifact(ctx, ver)
	if err != nil {
		return err
	}

	cfg.Printer.Printf("Downloaded bootstrap component %s with version %s\n",
		printer.BoldBlue(repositoryURL),
		printer.BoldBlue(ver))

	finalLocation := name
	if cfg.ExportPath != "" {
		baseName := path.Base(name)
		newLocation := path.Join(cfg.ExportPath, baseName)
		if err := os.Rename(name, newLocation); err != nil {
			return err
		}
		finalLocation = newLocation
	}

	cfg.Printer.Printf("Exported bootstrap component to %s\n", printer.BoldBlue(finalLocation))
	return nil
}
