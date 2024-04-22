// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ocm

import (
	"fmt"

	"github.com/gabriel-vasile/mimetype"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/open-component-model/ocm/cmds/ocm/commands/ocmcmds/common/inputs/types/file"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext/attrs/tmpcache"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/ociartifact"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/resourcetypes"
	"k8s.io/utils/pointer"
)

// from https://github.com/phoban01/gitops-component-cli/blob/main/pkg/component/handlers.go

type addFileOpts struct {
	name     string
	version  string
	path     string
	fileType string
}

func fileHandler(cv ocm.ComponentVersionAccess, octx ocm.Context, opts *addFileOpts) error {
	tmpcache.Set(octx, &tmpcache.Attribute{Path: "/tmp"})

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
			Name:    opts.name,
			Version: opts.version,
		},
		Relation: metav1.LocalRelation,
		Type:     ftype,
	}

	if err := cv.SetResourceBlob(r, acc, "", nil); err != nil {
		return err
	}

	return nil
}

type addImageOpts struct {
	name       string
	image      string
	version    string
	skipDigest bool
}

func imageHandler(cv ocm.ComponentVersionAccess, opts *addImageOpts) error {
	r := &compdesc.ResourceMeta{
		ElementMeta: compdesc.ElementMeta{
			Name:    opts.name,
			Version: opts.version,
		},
		Relation: metav1.ExternalRelation,
		Type:     resourcetypes.OCI_IMAGE,
	}

	spec := ociartifact.New(opts.image)

	modificationOptions := &ocm.ModificationOptions{
		ModifyResource: pointer.Bool(true),
	}
	if opts.skipDigest {
		modificationOptions.SkipDigest = pointer.Bool(opts.skipDigest)
	}

	if err := cv.SetResource(r, spec, modificationOptions); err != nil {
		return fmt.Errorf("failed to add image: %w", err)
	}

	return nil
}

type addReferenceOpts struct {
	name      string
	version   string
	component string
}

func referenceHandler(cv ocm.ComponentVersionAccess, opts *addReferenceOpts) error {
	r := &compdesc.ComponentReference{
		ElementMeta: compdesc.ElementMeta{
			Name:    opts.name,
			Version: opts.version,
		},
		ComponentName: opts.component,
	}

	if err := cv.SetReference(r); err != nil {
		return fmt.Errorf("failed to add reference: %w", err)
	}

	return nil
}
