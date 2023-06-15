// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ocm

import (
	"fmt"
	"net/url"
	"os"
	"strings"

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

type Options struct {
	Provider       string
	ProviderLabels metav1.Labels
	Labels         metav1.Labels
	ArchivePath    string
	RepositoryURL  string
	username       string
	token          string
}

type ComponentOption func(*Options)
type Component struct {
	Context clictx.Context
	Name    string
	Version string
	Options
}

func WithProvider(provider string) ComponentOption {
	return func(o *Options) {
		o.Provider = provider
	}
}

func WithProviderLabels(labels metav1.Labels) ComponentOption {
	return func(o *Options) {
		o.ProviderLabels = labels
	}
}

func WithLabels(labels metav1.Labels) ComponentOption {
	return func(o *Options) {
		o.Labels = labels
	}
}

func WithArchivePath(archivePath string) ComponentOption {
	return func(o *Options) {
		o.ArchivePath = archivePath
	}
}

func WithRepositoryURL(repositoryURL string) ComponentOption {
	return func(o *Options) {
		o.RepositoryURL = repositoryURL
	}
}

func WithUsername(username string) ComponentOption {
	return func(o *Options) {
		o.username = username
	}
}

func WithToken(token string) ComponentOption {
	return func(o *Options) {
		o.token = token
	}
}

func NewComponent(ctx clictx.Context, name, version string, opts ...ComponentOption) *Component {
	options := &Options{}
	for _, opt := range opts {
		opt(options)
	}
	return &Component{
		Context: ctx,
		Name:    name,
		Version: version,
		Options: *options,
	}
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
	Name          string
	Path          string
	Type          string
	InputType     string
	Version       string
	Image         string
	ComponentName string
}

type ResourceOption func(*ResourceOptions)

func WithComponentName(component string) ResourceOption {
	return func(o *ResourceOptions) {
		o.ComponentName = component
	}
}

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
	case "componentReference":
		o := &addReferenceOpts{
			name:      resOpt.Name,
			version:   resOpt.Version,
			component: resOpt.ComponentName,
		}
		if err := referenceHandler(arch, o); err != nil {
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

	regURL, err := parseURL(c.RepositoryURL)
	if err != nil {
		return err
	}

	meta := ocireg.NewComponentRepositoryMeta(strings.TrimPrefix(regURL.Path, "/"), ocireg.OCIRegistryURLPathMapping)
	targetSpec := ocireg.NewRepositorySpec(regURL.Host, meta)
	target, err := c.Context.OCMContext().RepositoryForSpec(targetSpec)
	if err != nil {
		return err
	}
	defer target.Close()

	handler, err := standard.New(
		standard.Recursive(true),
		standard.Overwrite(true),
		standard.Resolver(target))
	if err != nil {
		return err
	}

	// configure token
	err = c.configureCredentials(regURL.Host)
	if err != nil {
		return err
	}

	return transfer.TransferVersion(common.NewPrinter(c.Context.StdOut()), nil, arch, target, handler)
}

func (c *Component) configureCredentials(host string) error {
	consumerID := credentials.NewConsumerIdentity(identity.CONSUMER_TYPE,
		identity.ID_HOSTNAME, host,
		identity.ID_PATHPREFIX, c.username,
	)

	creds := credentials.DirectCredentials{
		credentials.ATTR_USERNAME:       c.username,
		credentials.ATTR_IDENTITY_TOKEN: c.token,
	}

	c.Context.OCMContext().CredentialsContext().SetCredentialsForConsumer(consumerID, creds)
	return nil
}

func parseURL(target string) (*url.URL, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url %s: %w", target, err)
	}
	if u.Host == "" {
		target = fmt.Sprintf("https://%s", target)
		u, err = url.Parse(target)
		if err != nil {
			return nil, fmt.Errorf("failed to parse url %s: %w", target, err)
		}
	}
	return u, nil
}
