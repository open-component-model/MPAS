// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ocm

import (
	"fmt"
	"net/url"
	"os"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/clictx"
	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/oci/identity"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/attrs/compatattr"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/comparch"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/ocireg"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/transfer"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/transfer/transferhandler/standard"
)

type Component struct {
	Context        clictx.Context
	Name           string
	Version        string
	Provider       string
	ProviderLabels metav1.Labels
	Labels         metav1.Labels
	ArchivePath    string
	RepositoryURL  string
}

func (c *Component) CreateComponentArchive(opts ...accessio.Option) error {
	obj, err := comparch.Create(c.Context.OCMContext(), accessobj.ACC_CREATE, c.ArchivePath, os.ModePerm, opts...)
	if err != nil {
		return err
	}

	desc := obj.GetDescriptor()
	desc.Name = c.Name
	desc.Version = c.Version
	desc.Provider.Name = metav1.ProviderName(c.Provider)
	desc.Provider.Labels = c.ProviderLabels
	desc.Labels = c.Labels
	if !compatattr.Get(c.Context) {
		desc.CreationTime = metav1.NewTimestampP()
	}

	err = compdesc.Validate(desc)
	if err != nil {
		obj.Close()
		os.RemoveAll(c.ArchivePath)
		return fmt.Errorf("invalid component info: %s", err)
	}
	err = obj.Close()
	if err != nil {
		os.RemoveAll(c.ArchivePath)
	}
	return err
}

type ResourceOptions struct {
	Name      string
	Path      string
	Type      string
	InputType string
	Version   string
	Image     string
}

type ResourceOption func(*ResourceOptions)

func WithResourceImage(image string) ResourceOption {
	return func(o *ResourceOptions) {
		o.Image = image
	}
}

func WithResourceName(name string) ResourceOption {
	return func(o *ResourceOptions) {
		o.Name = name
	}
}

func WithResourcePath(path string) ResourceOption {
	return func(o *ResourceOptions) {
		o.Path = path
	}
}

func WithResourceType(typ string) ResourceOption {
	return func(o *ResourceOptions) {
		o.Type = typ
	}
}

func WithResourceInputType(typ string) ResourceOption {
	return func(o *ResourceOptions) {
		o.InputType = typ
	}
}

func WithResourceVersion(version string) ResourceOption {
	return func(o *ResourceOptions) {
		o.Version = version
	}
}

func (c *Component) AddResource(username, token string, opts ...ResourceOption) error {
	resOpt := &ResourceOptions{}
	for _, opt := range opts {
		opt(resOpt)
	}
	if resOpt.Name == "" {
		return fmt.Errorf("resource name must be set")
	}
	arch, err := comparch.Open(c.Context.OCMContext(), accessobj.ACC_WRITABLE, c.ArchivePath, os.ModePerm)
	if err != nil {
		return err
	}
	defer arch.Close()
	switch resOpt.Type {
	case "file":
		if resOpt.Path == "" {
			return fmt.Errorf("resource path must be set")
		}
		o := &addFileOpts{
			name: resOpt.Name,
			path: resOpt.Path,
		}
		if err := fileHandler(arch, o); err != nil {
			return err
		}
	case "ociImage":
		o := &addImageOpts{
			name:    resOpt.Name,
			image:   resOpt.Image,
			version: resOpt.Version,
		}
		if err := imageHandler(arch, o); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported resource type: %s", resOpt.Type)
	}
	return nil
}

func (c *Component) Transfer(username, token string) error {
	session := ocm.NewSession(nil)
	defer session.Close()
	session.Finalize(c.Context.OCMContext())
	arch, err := comparch.Open(c.Context.OCMContext(), accessobj.ACC_READONLY, c.ArchivePath, 0)
	if err != nil {
		return err
	}
	session.Closer(arch)

	targetSpec := ocireg.NewRepositorySpec(c.RepositoryURL, nil)
	target, err := c.Context.OCMContext().RepositoryForSpec(targetSpec)
	if err != nil {
		return err
	}
	defer target.Close()

	handler, err := standard.New(
		standard.Recursive(true),
		standard.ResourcesByValue(true),
		standard.Overwrite(true),
		standard.Resolver(target))
	if err != nil {
		return err
	}

	// configure token
	err = c.configureCredentials(username, token)
	if err != nil {
		return err
	}

	return transfer.TransferVersion(common.NewPrinter(c.Context.StdOut()), nil, arch, target, handler)
}

func (c *Component) configureCredentials(username, token string) error {
	regURL, err := url.Parse(c.RepositoryURL)
	if err != nil {
		return err
	}

	consumerID := credentials.NewConsumerIdentity(identity.CONSUMER_TYPE,
		identity.ID_HOSTNAME, regURL.Host,
		identity.ID_PATHPREFIX, username,
	)

	creds := credentials.DirectCredentials{
		credentials.ATTR_USERNAME: username,
		credentials.ATTR_PASSWORD: token,
	}

	c.Context.OCMContext().CredentialsContext().SetCredentialsForConsumer(consumerID, creds)
	return nil
}
