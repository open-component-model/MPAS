// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ocm

import (
	"fmt"

	"github.com/gabriel-vasile/mimetype"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/open-component-model/ocm/cmds/ocm/commands/ocmcmds/common/inputs/types/file"
	"github.com/open-component-model/ocm/cmds/ocm/commands/ocmcmds/common/inputs/types/ociimage"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/clictx"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext/attrs/tmpcache"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/ociartifact"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/comparch"
)

// from https://github.com/phoban01/gitops-component-cli/blob/main/pkg/component/handlers.go

// addFileOpts contains the options for adding a file to a component archive.
type addFileOpts struct {
	name     string
	path     string
	fileType string
}

func fileHandler(c *comparch.ComponentArchive, opts *addFileOpts) error {
	tmpcache.Set(clictx.DefaultContext(), &tmpcache.Attribute{Path: "/tmp"})

	mtype, err := mimetype.DetectFile(opts.path)
	if err != nil {
		return err
	}

	ftype := file.TYPE
	if opts.fileType != "" {
		ftype = opts.fileType
	}

	fs := osfs.New()
	acc := accessio.BlobAccessForFile(mtype.String(), opts.path, fs)

	r := &compdesc.ResourceMeta{
		ElementMeta: compdesc.ElementMeta{
			Name: opts.name,
		},
		Relation: metav1.LocalRelation,
		Type:     ftype,
	}

	if err := c.SetResourceBlob(r, acc, "", nil); err != nil {
		return err
	}

	if err := c.Update(); err != nil {
		return err
	}

	return nil
}

type addImageOpts struct {
	name    string
	image   string
	version string
}

func imageHandler(c *comparch.ComponentArchive, opts *addImageOpts) error {
	r := &compdesc.ResourceMeta{
		ElementMeta: compdesc.ElementMeta{
			Name:    opts.name,
			Version: opts.version,
		},
		Relation: metav1.ExternalRelation,
		Type:     ociimage.TYPE,
	}

	spec := ociartifact.New(opts.image)

	if err := c.SetResource(r, spec); err != nil {
		return fmt.Errorf("failed to add image: %w", err)
	}

	if err := c.Update(); err != nil {
		return fmt.Errorf("failed to update component archive: %w", err)
	}

	return nil
}

type addReferenceOpts struct {
	name      string
	version   string
	component string
}

func referenceHandler(c *comparch.ComponentArchive, opts *addReferenceOpts) error {
	r := &compdesc.ComponentReference{
		ElementMeta: compdesc.ElementMeta{
			Name:    opts.name,
			Version: opts.version,
		},
		ComponentName: opts.component,
	}

	if err := c.SetReference(r); err != nil {
		return fmt.Errorf("failed to add reference: %w", err)
	}

	if err := c.Update(); err != nil {
		return fmt.Errorf("failed to update component archive: %w", err)
	}

	return nil
}
