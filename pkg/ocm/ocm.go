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
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/attrs/compatattr"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/comparch"
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
}

type ResourceOption func(*ResourceOptions)

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

func (c *Component) AddResource(opts ...ResourceOption) error {
	resOpt := &ResourceOptions{}
	for _, opt := range opts {
		opt(resOpt)
	}
	if resOpt.Name == "" || resOpt.Path == "" {
		return fmt.Errorf("resource name must be set")
	}
	arch, err := comparch.Open(c.Context.OCMContext(), accessobj.ACC_WRITABLE, c.ArchivePath, os.ModePerm)
	if err != nil {
		return err
	}
	defer arch.Close()
	switch resOpt.Type {
	case "file":
		o := &addFileOpts{
			name: resOpt.Name,
			path: resOpt.Path,
		}
		if err := fileHandler(arch, o); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported resource type: %s", resOpt.Type)
	}
	return nil
}

func (c *Component) Transfer() error {
	session := ocm.NewSession(nil)
	defer session.Close()
	session.Finalize(c.Context.OCMContext())
	arch, err := comparch.Open(c.Context.OCMContext(), accessobj.ACC_READONLY, c.ArchivePath, 0)
	if err != nil {
		return err
	}
	session.Closer(arch)
	target, err := ocm.AssureTargetRepository(session, c.Context.OCMContext(), c.RepositoryURL, ocm.CommonTransportFormat)
	if err != nil {
		return err
	}

	handler, err := standard.New()
	if err != nil {
		return err
	}

	return transfer.TransferVersion(common.NewPrinter(c.Context.StdOut()), nil, arch, target, handler)
}

func (c *Component) ConfigureCredentials(token string) error {
	regURL, err := url.Parse(c.RepositoryURL)
	if err != nil {
		return err
	}

	if regURL.Scheme == "" {
		regURL, err = url.Parse(fmt.Sprintf("oci://%s", c.RepositoryURL))
		if err != nil {
			return err
		}
	}

	consumerID := credentials.ConsumerIdentity{
		"type":     "OCIRegistry",
		"hostname": regURL.Host,
	}

	props := make(common.Properties)
	props.SetNonEmptyValue("token", token)
	creds := credentials.NewCredentials(props)
	c.Context.OCMContext().CredentialsContext().SetCredentialsForConsumer(consumerID, creds)
	return nil
}
