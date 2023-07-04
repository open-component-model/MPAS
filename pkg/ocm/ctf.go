// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ocm

import (
	"fmt"
	"os"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/ctf"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/transfer"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/transfer/transferhandler/standard"
	"github.com/open-component-model/ocm/pkg/finalizer"
)

type CTF ocm.Repository

// CreateCTF creates a new ctf repository.
func CreateCTF(ctx ocm.Context, repoPath string, opts ...accessio.Option) (CTF, error) {
	ctf, err := ctf.Open(ctx, accessobj.ACC_CREATE, repoPath, os.ModePerm, opts...)
	if err != nil {
		return nil, err
	}

	return ctf, nil
}

// Transfer transfers a component to a repository.
// It accepts a target corresponding to the repository.
func Transfer(octx ocm.Context, repo, target ocm.Repository) (rerr error) {
	var finalize finalizer.Finalizer
	defer finalize.FinalizeWithErrorPropagation(&rerr)

	lister := repo.ComponentLister()
	if lister == nil {
		return fmt.Errorf("repo does not support lister")
	}
	comps, err := lister.GetComponents("", true)
	if rerr != nil {
		return fmt.Errorf("failed to list components: %w", err)
	}

	printer := common.NewPrinter(os.Stdout)
	closure := transfer.TransportClosure{}
	transferHandler, err := standard.New(standard.Overwrite())
	if err != nil {
		return err
	}
	for _, cname := range comps {
		loop := finalize.Nested()

		c, err := repo.LookupComponent(cname)
		if err != nil {
			return fmt.Errorf("cannot get component %s", cname)
		}
		loop.Close(c)

		vnames, err := c.ListVersions()
		if err != nil {
			return fmt.Errorf("cannot list versions for component %s", cname)
		}

		for _, vname := range vnames {
			loop := loop.Nested()

			cv, err := c.LookupVersion(vname)
			if err != nil {
				return fmt.Errorf("cannot get version %s for component %s", vname, cname)
			}
			loop.Close(cv)

			err = transfer.TransferVersion(printer, closure, cv, target, transferHandler)
			if err != nil {
				return fmt.Errorf("cannot transfer version %s for component %s: %w", vname, cname, err)
			}

			if err := loop.Finalize(); err != nil {
				return err
			}
		}
		if err := loop.Finalize(); err != nil {
			return err
		}
	}

	return nil
}
