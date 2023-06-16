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
	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/oci/identity"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	om "github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/attrs/compatattr"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/comparch"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/transfer"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/transfer/transferhandler/standard"
)

// options contains the options for creating a component archive.
type options struct {
	provider       string
	providerLabels metav1.Labels
	labels         metav1.Labels
	archivePath    string
	repositoryURL  string
	username       string
	token          string
}

// ComponentOption is a function that configures a component options.
type ComponentOption func(*options)

// Component contains the information for managing a component.
// It is used to create a component archive, add resources and transfer the component
// to a repository.
type Component struct {
	// Context is the context used for creating the component.
	Context om.Context
	// Name is the name of the component.
	Name string
	// Version is the version of the component.
	Version string
	options
}

// WithProvider configures the provider of the component.
func WithProvider(provider string) ComponentOption {
	return func(o *options) {
		o.provider = provider
	}
}

// WithProviderLabels configures the provider labels of the component.
func WithProviderLabels(labels metav1.Labels) ComponentOption {
	return func(o *options) {
		o.providerLabels = labels
	}
}

// WithLabels configures the labels of the component.
func WithLabels(labels metav1.Labels) ComponentOption {
	return func(o *options) {
		o.labels = labels
	}
}

// WithArchivePath configures the archive path of the component.
func WithArchivePath(archivePath string) ComponentOption {
	return func(o *options) {
		o.archivePath = archivePath
	}
}

// WithRepositoryURL configures the repository url of the component.
func WithRepositoryURL(repositoryURL string) ComponentOption {
	return func(o *options) {
		o.repositoryURL = repositoryURL
	}
}

// WithUsername configures the username of the component.
func WithUsername(username string) ComponentOption {
	return func(o *options) {
		o.username = username
	}
}

// WithToken configures the token of the component.
func WithToken(token string) ComponentOption {
	return func(o *options) {
		o.token = token
	}
}

// NewComponent creates a new component.
func NewComponent(ctx om.Context, name, version string, opts ...ComponentOption) *Component {
	options := &options{}
	for _, opt := range opts {
		opt(options)
	}
	return &Component{
		Context: ctx,
		Name:    name,
		Version: version,
		options: *options,
	}
}

// CreateComponentArchive creates a component archive.
// It accepts options for configuring the component archive.
func (c *Component) CreateComponentArchive(opts ...accessio.Option) error {
	obj, err := comparch.Create(c.Context, accessobj.ACC_CREATE, c.archivePath, os.ModePerm, opts...)
	if err != nil {
		return err
	}

	desc := obj.GetDescriptor()
	desc.Name = c.Name
	desc.Version = c.Version
	desc.Provider.Name = metav1.ProviderName(c.provider)
	desc.Provider.Labels = c.providerLabels
	desc.Labels = c.labels
	if !compatattr.Get(c.Context) {
		desc.CreationTime = metav1.NewTimestampP()
	}

	err = compdesc.Validate(desc)
	if err != nil {
		obj.Close()
		os.RemoveAll(c.archivePath)
		return fmt.Errorf("invalid component info: %s", err)
	}
	err = obj.Close()
	if err != nil {
		os.RemoveAll(c.archivePath)
	}
	return err
}

// ResourceOptions contains the options for adding a resource to a component archive.
type ResourceOptions struct {
	name          string
	path          string
	typ           string
	inputType     string
	version       string
	image         string
	componentName string
}

// ResourceOption is a function that configures a resource options.
type ResourceOption func(*ResourceOptions)

// WithComponentName configures the component name of the resource to be added.
func WithComponentName(component string) ResourceOption {
	return func(o *ResourceOptions) {
		o.componentName = component
	}
}

// WithResourceImage configures the image of the resource.
func WithResourceImage(image string) ResourceOption {
	return func(o *ResourceOptions) {
		o.image = image
	}
}

// WithResourceName configures the name of the resource.
func WithResourceName(name string) ResourceOption {
	return func(o *ResourceOptions) {
		o.name = name
	}
}

// WithResourcePath configures the path of the resource.
func WithResourcePath(path string) ResourceOption {
	return func(o *ResourceOptions) {
		o.path = path
	}
}

// WithResourceType configures the type of the resource.
func WithResourceType(typ string) ResourceOption {
	return func(o *ResourceOptions) {
		o.typ = typ
	}
}

// WithResourceInputType configures the input type of the resource.
func WithResourceInputType(typ string) ResourceOption {
	return func(o *ResourceOptions) {
		o.inputType = typ
	}
}

// WithResourceVersion configures the version of the resource.
func WithResourceVersion(version string) ResourceOption {
	return func(o *ResourceOptions) {
		o.version = version
	}
}

// AddResource adds a resource to a component archive.
// It accepts options for configuring the resource.
// The resource type can be one of the following:
// - file
// - ociImage
// - componentReference
func (c *Component) AddResource(opts ...ResourceOption) error {
	resOpt := &ResourceOptions{}
	for _, opt := range opts {
		opt(resOpt)
	}
	if resOpt.name == "" {
		return fmt.Errorf("resource name must be set")
	}
	arch, err := comparch.Open(c.Context, accessobj.ACC_WRITABLE, c.archivePath, os.ModePerm)
	if err != nil {
		return err
	}
	defer arch.Close()
	switch resOpt.typ {
	case "file":
		if resOpt.path == "" {
			return fmt.Errorf("resource path must be set")
		}
		o := &addFileOpts{
			name:    resOpt.name,
			path:    resOpt.path,
			version: resOpt.version,
		}
		if err := fileHandler(arch, c.Context, o); err != nil {
			return err
		}
	case "ociImage":
		o := &addImageOpts{
			name:    resOpt.name,
			image:   resOpt.image,
			version: resOpt.version,
		}
		if err := imageHandler(arch, o); err != nil {
			return err
		}
	case "componentReference":
		o := &addReferenceOpts{
			name:      resOpt.name,
			version:   resOpt.version,
			component: resOpt.componentName,
		}
		if err := referenceHandler(arch, o); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported resource type: %s", resOpt.typ)
	}
	return nil
}

// Transfer transfers a component to a repository.
// It accepts a target corresponding to the repository.
func (c *Component) Transfer(target om.Repository) error {
	session := ocm.NewSession(nil)
	defer session.Close()
	session.Finalize(c.Context)
	arch, err := comparch.Open(c.Context, accessobj.ACC_READONLY, c.archivePath, 0)
	if err != nil {
		return err
	}
	session.Closer(arch)

	regURL, err := ParseURL(c.repositoryURL)
	if err != nil {
		return err
	}

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

	return transfer.TransferVersion(common.NewPrinter(os.Stdout), nil, arch, target, handler)
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

	c.Context.CredentialsContext().SetCredentialsForConsumer(consumerID, creds)
	return nil
}

// ParseURL parses a url and adds the scheme if missing.
// It returns an error if the url is invalid.
func ParseURL(target string) (*url.URL, error) {
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
