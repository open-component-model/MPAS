// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ocm

import (
	"fmt"

	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/attrs/compatattr"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/finalizer"
)

// options contains the options for creating a component archive.
type options struct {
	provider       string
	providerLabels metav1.Labels
	labels         metav1.Labels
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
	Context ocm.Context
	// Name is the name of the component.
	Name string
	// Version is the version of the component.
	Version string
	// access is the component access.
	access ocm.ComponentAccess
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
func NewComponent(ctx ocm.Context, name, version string, opts ...ComponentOption) (*Component, error) {
	options := &options{}
	for _, opt := range opts {
		opt(options)
	}

	c := &Component{
		Context: ctx,
		Name:    name,
		Version: version,
		options: *options,
	}

	if err := c.configureCredentials(); err != nil {
		return nil, err
	}

	return c, nil
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
	skipDigest    bool
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

// WithSkipVerify adds an option to skip the digest calculation of this resource.
func WithSkipVerify(skip bool) ResourceOption {
	return func(o *ResourceOptions) {
		o.skipDigest = skip
	}
}

// AddResource adds a resource to a component archive.
// It accepts options for configuring the resource.
// The resource type can be one of the following:
// - file
// - ociImage
// - componentReference
func (c *Component) AddResource(opts ...ResourceOption) (rerr error) {
	var finalize finalizer.Finalizer
	defer finalize.FinalizeWithErrorPropagation(&rerr)
	resOpt := &ResourceOptions{}
	for _, opt := range opts {
		opt(resOpt)
	}
	if resOpt.name == "" {
		return fmt.Errorf("resource name must be set")
	}
	cv, err := c.access.LookupVersion(c.Version)
	if err != nil {
		if err != nil {
			return err
		}
	}
	defer finalize.Close(cv)
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
		if err := fileHandler(cv, c.Context, o); err != nil {
			return err
		}
	case "ociImage":
		o := &addImageOpts{
			name:       resOpt.name,
			image:      resOpt.image,
			version:    resOpt.version,
			skipDigest: resOpt.skipDigest,
		}
		if err := imageHandler(cv, o); err != nil {
			return err
		}
	case "componentReference":
		o := &addReferenceOpts{
			name:      resOpt.name,
			version:   resOpt.version,
			component: resOpt.componentName,
		}
		if err := referenceHandler(cv, o); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported resource type: %s", resOpt.typ)
	}

	if err := c.access.AddVersion(cv); err != nil {
		return err
	}
	return nil
}

// AddToCTF adds a component to a ctf repository.
func (c *Component) AddToCTF(repo ocm.Repository) (rerr error) {
	var finalize finalizer.Finalizer
	defer finalize.FinalizeWithErrorPropagation(&rerr)

	var err error
	c.access, err = repo.LookupComponent(c.Name)
	if err != nil {
		return err
	}

	cv, err := c.access.LookupVersion(c.Version)
	if err != nil {
		cv, err = c.access.NewVersion(c.Version)
		if err != nil {
			return fmt.Errorf("failed to create component version %s: %w", c.Version, err)
		}
	}
	finalize.Close(cv)

	desc := cv.GetDescriptor()
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
		return fmt.Errorf("cannot add component to ctf: %s", err)
	}

	err = c.access.AddVersion(cv)
	if err != nil {
		return fmt.Errorf("cannot add component to ctf: %s", err)
	}
	return nil
}

// Close closes the component access.
func (c *Component) Close() error {
	return c.access.Close()
}
