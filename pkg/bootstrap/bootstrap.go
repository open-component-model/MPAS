// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/open-component-model/mpas/pkg/env"
	"github.com/open-component-model/mpas/pkg/kubeutils"
	"github.com/open-component-model/mpas/pkg/ocm"
	"github.com/open-component-model/mpas/pkg/printer"
	om "github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	errReconciledWithWarning = errors.New("reconciled with warning")
)

// options contains the options to be used during bootstrap
type options struct {
	description           string
	defaultBranch         string
	visibility            string
	personal              bool
	owner                 string
	token                 string
	repositoryName        string
	targetPath            string
	commitMessageAppendix string
	fromFile              string
	registry              string
	dockerConfigPath      string
	transportType         string
	kubeclient            client.Client
	restClientGetter      genericclioptions.RESTClientGetter
	components            []string
	interval              time.Duration
	timeout               time.Duration
	printer               *printer.Printer
	testURL               string
}

// Option is a function that sets an option on the bootstrap
type Option func(*options)

// Bootstrap runs the bootstrap of mpas.
// This means it creates a new management repository and the installs the bootstrap component
// in the cluster targeted by the kubeconfig.
type Bootstrap struct {
	providerClient gitprovider.Client
	repository     gitprovider.UserRepository
	url            string
	options
}

// WithTestURL sets the testURL to use for the bootstrap component
func WithTestURL(testURL string) Option {
	return func(o *options) {
		o.testURL = testURL
	}
}

// WithCommitMessageAppendix sets the commit message appendix to use for the bootstrap component
func WithCommitMessageAppendix(commitMessageAppendix string) Option {
	return func(o *options) {
		o.commitMessageAppendix = commitMessageAppendix
	}
}

// WithInterval sets the interval to use for the bootstrap component
func WithInterval(interval time.Duration) Option {
	return func(o *options) {
		o.interval = interval
	}
}

// WithTimeout sets the timeout to use for the bootstrap component
func WithTimeout(timeout time.Duration) Option {
	return func(o *options) {
		o.timeout = timeout
	}
}

// WithRESTClientGetter sets the RESTClientGetter to use for the bootstrap component
func WithRESTClientGetter(restClientGetter genericclioptions.RESTClientGetter) Option {
	return func(o *options) {
		o.restClientGetter = restClientGetter
	}
}

// WithKubeClient sets the kubeclient to use for the bootstrap component
func WithKubeClient(kubeclient client.Client) Option {
	return func(o *options) {
		o.kubeclient = kubeclient
	}
}

// WithDockerConfigPath sets the docker config path to use for the bootstrap component
func WithDockerConfigPath(dockerConfigPath string) Option {
	return func(o *options) {
		o.dockerConfigPath = dockerConfigPath
	}
}

// WithTarget sets the targetPath of the bootstrap component
func WithTarget(targetPath string) Option {
	return func(o *options) {
		targetPath = strings.TrimSuffix(targetPath, "/")
		o.targetPath = targetPath
	}
}

// WithPrinter sets the printer to use for printing messages
func WithPrinter(printer *printer.Printer) Option {
	return func(o *options) {
		o.printer = printer
	}
}

// WithComponents sets the components to include in the management repository
func WithComponents(components []string) Option {
	return func(o *options) {
		o.components = components
	}
}

// WithDescription sets the description of the management repository
func WithDescription(description string) Option {
	return func(o *options) {
		o.description = description
	}
}

// WithDefaultBranch sets the default branch of the management repository
func WithDefaultBranch(defaultBranch string) Option {
	return func(o *options) {
		o.defaultBranch = defaultBranch
	}
}

// WithVisibility sets the visibility of the management repository
func WithVisibility(visibility string) Option {
	return func(o *options) {
		o.visibility = visibility
	}
}

// WithPersonal sets the personal flag of the management repository
func WithPersonal(personal bool) Option {
	return func(o *options) {
		o.personal = personal
	}
}

// WithOwner sets the owner of the management repository
func WithOwner(owner string) Option {
	return func(o *options) {
		o.owner = owner
	}
}

// WithToken sets the token of the management repository
func WithToken(token string) Option {
	return func(o *options) {
		o.token = token
	}
}

// WithRepositoryName sets the repository name of the management repository
func WithRepositoryName(repositoryName string) Option {
	return func(o *options) {
		o.repositoryName = repositoryName
	}
}

// WithFromFile sets the file from which to read the bootstrap component
func WithFromFile(fromFile string) Option {
	return func(o *options) {
		o.fromFile = fromFile
	}
}

// WithRegistry sets the registry to use for the bootstrap component
func WithRegistry(registry string) Option {
	return func(o *options) {
		o.registry = registry
	}
}

// WithTransportType sets the transport type to use for git operations
func WithTransportType(transportType string) Option {
	return func(o *options) {
		o.transportType = transportType
	}
}

// New returns a new Bootstrap. It accepts a gitprovider.Client and a list of options.
func New(providerClient gitprovider.Client, opts ...Option) (*Bootstrap, error) {
	b := &Bootstrap{
		providerClient: providerClient,
	}

	for _, opt := range opts {
		opt(&b.options)
	}

	setDefaults(b)

	if err := validateOptions(&b.options); err != nil {
		return nil, err
	}

	return b, nil
}

// Run runs the bootstrap of mpas and returns an error if it fails.
func (b *Bootstrap) Run(ctx context.Context) error {
	if b.fromFile != "" {
		return fmt.Errorf("bootstrap from file is not supported yet")
	}

	b.printer.Printf("Running %s ...\n",
		printer.BoldBlue("mpas bootstrap"))

	if err := b.printer.PrintSpinner(fmt.Sprintf("Preparing Management repository %s",
		printer.BoldBlue(b.repositoryName))); err != nil {
		return err
	}

	if err := b.reconcileManagementRepository(ctx); err != nil {
		if er := b.printer.StopFailSpinner(fmt.Sprintf("Preparing Management repository %s with branch %s and visibility %s",
			printer.BoldBlue(b.repositoryName),
			printer.BoldBlue(b.defaultBranch),
			printer.BoldBlue(b.visibility))); er != nil {
			err = errors.Join(err, er)
		}
		return err
	}

	if err := b.printer.StopSpinner(fmt.Sprintf("Preparing Management repository %s with branch %s and visibility %s",
		printer.BoldBlue(b.repositoryName),
		printer.BoldBlue(b.defaultBranch),
		printer.BoldBlue(b.visibility))); err != nil {
		return err
	}

	ociRepo, err := ocm.MakeRepositoryWithDockerConfig(b.registry, b.dockerConfigPath)
	if err != nil {
		return err
	}
	defer ociRepo.Close()

	if err = b.printer.PrintSpinner(fmt.Sprintf("Fetching bootstrap component from %s ",
		printer.BoldBlue(b.registry))); err != nil {
		return err
	}
	refs, err := b.fetchBootstrapComponentReferences(ociRepo)
	if err != nil {
		if er := b.printer.StopFailSpinner(fmt.Sprintf("Fetching bootstrap component from %s ",
			printer.BoldBlue(b.registry))); er != nil {
			err = errors.Join(err, er)
		}
		return fmt.Errorf("failed to fetch bootstrap component references: %w", err)
	}

	if err := b.printer.StopSpinner(fmt.Sprintf("Fetching bootstrap component from %s ",
		printer.BoldBlue(b.registry))); err != nil {
		return err
	}

	fluxRef, ok := refs[env.FluxName]
	if !ok {
		return fmt.Errorf("flux component not found")
	}

	err = b.printer.PrintSpinner(fmt.Sprintf("Installing %s with version %s",
		printer.BoldBlue(env.FluxName),
		printer.BoldBlue(fluxRef.GetVersion())))
	if err != nil {
		return err
	}
	if err = b.installFlux(ctx, ociRepo, fluxRef); err != nil {
		if er := b.printer.StopFailSpinner(fmt.Sprintf("Installing %s with version %s",
			printer.BoldBlue(env.FluxName),
			printer.BoldBlue(fluxRef.GetVersion()))); er != nil {
			err = errors.Join(err, er)
		}
		return fmt.Errorf("failed to install flux: %w", err)
	}

	if err := b.printer.StopSpinner(fmt.Sprintf("Installing %s with version %s",
		printer.BoldBlue(env.FluxName),
		printer.BoldBlue(fluxRef.GetVersion()))); err != nil {
		return err
	}
	delete(refs, env.FluxName)

	compNs := make(map[string][]string)
	// install components in order by using the ordered keys
	comps := getOrderedKeys(refs)
	var latestSHA string
	for _, comp := range comps {
		ref := refs[comp]
		err := b.printer.PrintSpinner(fmt.Sprintf("Generating %s manifest with version %s",
			printer.BoldBlue(comp),
			printer.BoldBlue(ref.GetVersion())))
		if err != nil {
			return err
		}

		switch comp {
		case env.OcmControllerName:
			sha, err := b.installComponent(ctx, ociRepo, ref, comp, "ocm-system", compNs)
			if err != nil {
				return err
			}
			latestSHA = sha
			compNs["ocm-system"] = append(compNs["ocm-system"], comp)
		case "git-controller":
			sha, err := b.installComponent(ctx, ociRepo, ref, comp, "ocm-system", compNs)
			if err != nil {
				return err
			}
			latestSHA = sha
			compNs["ocm-system"] = append(compNs["ocm-system"], comp)
		case env.ReplicationControllerName:
			sha, err := b.installComponent(ctx, ociRepo, ref, comp, "ocm-system", compNs)
			if err != nil {
				return err
			}
			latestSHA = sha
			compNs["ocm-system"] = append(compNs["ocm-system"], comp)
		case env.MpasProductControllerName:
			sha, err := b.installComponent(ctx, ociRepo, ref, comp, "mpas-system", compNs)
			if err != nil {
				return err
			}
			latestSHA = sha
			compNs["mpas-system"] = append(compNs["mpas-system"], comp)
		case env.MpasProjectControllerName:
			sha, err := b.installComponent(ctx, ociRepo, ref, comp, "mpas-system", compNs)
			if err != nil {
				return err
			}
			latestSHA = sha
			compNs["mpas-system"] = append(compNs["mpas-system"], comp)
		default:
			err := fmt.Errorf("unknown component %q", comp)
			if er := b.printer.StopFailSpinner(fmt.Sprintf("Generating %s manifest with version %s",
				printer.BoldBlue(comp),
				printer.BoldBlue(ref.GetVersion()))); er != nil {
				err = errors.Join(err, er)
			}
			return err
		}

		if err := b.printer.StopSpinner(fmt.Sprintf("Generating %s manifest with version %s",
			printer.BoldBlue(comp),
			printer.BoldBlue(ref.GetVersion()))); err != nil {
			return err
		}
	}

	err = b.printer.PrintSpinner("Waiting for components to be ready")
	if err != nil {
		return err
	}

	expectedRevision := fmt.Sprintf("%s@sha1:%s", b.defaultBranch, latestSHA)
	if err := kubeutils.ReconcileGitrepository(ctx, b.kubeclient, env.DefaultFluxNamespace, env.DefaultFluxNamespace); err != nil {
		if er := b.printer.StopFailSpinner("Waiting for components to be ready"); er != nil {
			err = errors.Join(err, er)
		}
		return err
	}

	if err := kubeutils.ReportGitrepositoryHealth(ctx, b.kubeclient, env.DefaultFluxNamespace, env.DefaultFluxNamespace, expectedRevision, env.DefaultPollInterval, b.timeout); err != nil {
		if er := b.printer.StopFailSpinner("Waiting for components to be ready"); er != nil {
			err = errors.Join(err, er)
		}
		return fmt.Errorf("failed to report gitrepository health: %w", err)
	}

	if err := kubeutils.ReconcileKustomization(ctx, b.kubeclient, env.DefaultFluxNamespace, env.DefaultFluxNamespace); err != nil {
		if er := b.printer.StopFailSpinner("Waiting for components to be ready"); er != nil {
			err = errors.Join(err, er)
		}
		return err
	}

	if err := kubeutils.ReportKustomizationHealth(ctx, b.kubeclient, env.DefaultFluxNamespace, env.DefaultFluxNamespace, expectedRevision, env.DefaultPollInterval, b.timeout); err != nil {
		if er := b.printer.StopFailSpinner("Waiting for components to be ready"); er != nil {
			err = errors.Join(err, er)
		}
		return fmt.Errorf("failed to report kustomization health: %w", err)
	}
	for ns, comps := range compNs {
		if err := kubeutils.ReportComponentsHealth(ctx, b.restClientGetter, b.timeout, comps, ns); err != nil {
			if er := b.printer.StopFailSpinner("Waiting for components to be ready"); er != nil {
				err = errors.Join(err, er)
			}
			return fmt.Errorf("failed to report health, please try again in a few minutes: %w", err)
		}
	}
	if err := b.printer.StopSpinner("Waiting for components to be ready"); err != nil {
		return err
	}
	b.printer.Printf("\n")
	b.printer.Printf("Bootstrap completed successfully!\n")

	return nil
}

func (b *Bootstrap) installComponent(ctx context.Context, ociRepo om.Repository, ref compdesc.ComponentReference, comp, ns string, compNs map[string][]string) (string, error) {
	dir, err := mkdirTempDir(fmt.Sprintf("%s-install", comp))
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	opts := &componentOptions{
		kubeClient:            b.kubeclient,
		restClientGetter:      b.restClientGetter,
		gitRepository:         b.repository,
		branch:                b.defaultBranch,
		targetPath:            b.targetPath,
		commitMessageAppendix: b.commitMessageAppendix,
		namespace:             ns,
		provider:              string(b.providerClient.ProviderID()),
		dir:                   dir,
		timeout:               b.timeout,
		installedNS:           compNs,
	}

	inst, err := newComponentInstall(ref.GetComponentName(), ref.GetVersion(), ociRepo, opts)
	if err != nil {
		return "", err
	}
	sha, err := inst.install(ctx, fmt.Sprintf("%s-file", comp))
	if err != nil {
		if er := b.printer.StopFailSpinner(fmt.Sprintf("Generating %s manifest with version %s",
			printer.BoldBlue(comp),
			printer.BoldBlue(ref.GetVersion()))); er != nil {
			err = errors.Join(err, er)
		}
		return "", err
	}
	return sha, nil
}

func (b *Bootstrap) installFlux(ctx context.Context, ociRepo om.Repository, ref compdesc.ComponentReference) error {
	dir, err := mkdirTempDir("flux-install")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	opts := &fluxOptions{
		kubeClient:            b.kubeclient,
		restClientGetter:      b.restClientGetter,
		url:                   b.url,
		testURL:               b.testURL,
		transport:             b.transportType,
		branch:                b.defaultBranch,
		targetPath:            b.targetPath,
		commitMessageAppendix: b.commitMessageAppendix,
		dir:                   dir,
		interval:              b.interval,
		timeout:               b.timeout,
		token:                 b.token,
		namespace:             env.DefaultFluxNamespace,
	}
	inst, err := newFluxInstall(ref.GetComponentName(), ref.GetVersion(), b.owner, ociRepo, opts)
	if err != nil {
		return err
	}
	if err := inst.Install(ctx, "flux"); err != nil {
		return err
	}
	return nil
}

func (b *Bootstrap) fetchBootstrapComponentReferences(ociRepo om.Repository) (map[string]compdesc.ComponentReference, error) {
	cv, err := ocm.FetchLatestComponent(ociRepo, env.DefaultBootstrapComponent)
	if err != nil {
		return nil, err
	}

	return ocm.FetchComponenReferences(cv, b.components)
}

// reconcileManagementRepository reconciles the management repository. It creates it if it does not exist.
func (b *Bootstrap) reconcileManagementRepository(ctx context.Context) error {
	repo, err := b.reconcileRepository(ctx, b.personal)
	if err != nil && !errors.Is(err, errReconciledWithWarning) {
		return err
	}

	cloneURL, err := b.getCloneURL(repo, gitprovider.TransportTypeHTTPS)
	if err != nil {
		return err
	}

	b.repository = repo
	b.url = cloneURL

	return nil
}

// DeleteManagementRepository deletes the management repository.
func (b *Bootstrap) DeleteManagementRepository(ctx context.Context) error {
	if b.repository == nil {
		return fmt.Errorf("management repository is not set")
	}

	err := b.repository.Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete management repository: %w", err)
	}
	return nil
}

func (b *Bootstrap) reconcileRepository(ctx context.Context, personal bool) (gitprovider.UserRepository, error) {
	var (
		repo gitprovider.UserRepository
		err  error
	)
	subOrgs, repoName := splitSubOrganizationsFromRepositoryName(b.repositoryName)
	if personal {
		userRef := newUserRef(b.providerClient.SupportedDomain(), b.owner)
		repoRef := newUserRepositoryRef(userRef, repoName)
		repoInfo := newRepositoryInfo(b.description, b.defaultBranch, b.visibility)
		repo, err = b.providerClient.UserRepositories().Get(ctx, repoRef)
		if err != nil {
			if !errors.Is(err, gitprovider.ErrNotFound) {
				return nil, fmt.Errorf("failed to get Git repository %q: %w", repoRef.String(), err)
			}
			repo, _, err = b.providerClient.UserRepositories().Reconcile(ctx, repoRef, repoInfo)
			if err != nil {
				return nil, fmt.Errorf("failed to reconcile Git repository %q: %w", repoRef.String(), err)
			}
		}
	} else {
		orgRef, err := b.getOrganization(ctx, subOrgs)
		if err != nil {
			return nil, fmt.Errorf("failed to reconcile Git repository %q: %w", b.repositoryName, err)
		}
		repoRef := newOrgRepositoryRef(*orgRef, repoName)
		repoInfo := newRepositoryInfo(b.description, b.defaultBranch, b.visibility)
		repo, err = b.providerClient.OrgRepositories().Get(ctx, repoRef)
		if err != nil {
			if !errors.Is(err, gitprovider.ErrNotFound) {
				return nil, fmt.Errorf("failed to get Git repository %q: %w", repoRef.String(), err)
			}
			repo, _, err = b.providerClient.OrgRepositories().Reconcile(ctx, repoRef, repoInfo)
			if err != nil {
				return nil, fmt.Errorf("failed to create new Git repository %q: %w", repoRef.String(), err)
			}
		}
	}

	return repo, nil
}

func (b *Bootstrap) getOrganization(ctx context.Context, subOrgs []string) (*gitprovider.OrganizationRef, error) {
	return &gitprovider.OrganizationRef{
		Domain:           b.providerClient.SupportedDomain(),
		Organization:     b.owner,
		SubOrganizations: subOrgs,
	}, nil
}

func (b *Bootstrap) getCloneURL(repository gitprovider.UserRepository, transport gitprovider.TransportType) (string, error) {
	var url string
	if cloner, ok := repository.(gitprovider.CloneableURL); ok {
		url = cloner.GetCloneURL("", transport)
	} else {
		url = repository.Repository().GetCloneURL(transport)
	}

	if transport == gitprovider.TransportTypeSSH {
		return "", fmt.Errorf("SSH transport is not supported")
	}

	return url, nil
}

func splitSubOrganizationsFromRepositoryName(name string) ([]string, string) {
	elements := strings.Split(name, "/")
	switch i := len(elements); i {
	case 1:
		return nil, name
	default:
		return elements[:i-1], elements[i-1]
	}
}

func newOrgRepositoryRef(organizationRef gitprovider.OrganizationRef, name string) gitprovider.OrgRepositoryRef {
	return gitprovider.OrgRepositoryRef{
		OrganizationRef: organizationRef,
		RepositoryName:  name,
	}
}

func newUserRef(domain, login string) gitprovider.UserRef {
	return gitprovider.UserRef{
		Domain:    domain,
		UserLogin: login,
	}
}

func newUserRepositoryRef(userRef gitprovider.UserRef, name string) gitprovider.UserRepositoryRef {
	return gitprovider.UserRepositoryRef{
		UserRef:        userRef,
		RepositoryName: name,
	}
}

func newRepositoryInfo(description, defaultBranch, visibility string) gitprovider.RepositoryInfo {
	var i gitprovider.RepositoryInfo
	if description != "" {
		i.Description = gitprovider.StringVar(description)
	}
	if defaultBranch != "" {
		i.DefaultBranch = gitprovider.StringVar(defaultBranch)
	}
	if visibility != "" {
		i.Visibility = gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibility(visibility))
	}
	return i
}

func setDefaults(b *Bootstrap) {
	if b.description == "" {
		b.description = "Management repository for the Open Component Model"
	}

	if b.defaultBranch == "" {
		b.defaultBranch = "main"
	}

	if b.visibility == "" {
		b.visibility = "private"
	}

	if b.transportType == "" {
		b.transportType = "https"
	}
}

func validateOptions(opts *options) error {
	if opts.repositoryName == "" {
		return fmt.Errorf("repository name must be set")
	}

	if opts.restClientGetter == nil {
		return fmt.Errorf("rest client getter must be set")
	}

	if opts.kubeclient == nil {
		return fmt.Errorf("kubeclient must be set")
	}

	if opts.printer == nil {
		return fmt.Errorf("printer must be set")
	}

	return nil
}

func mkdirTempDir(pattern string) (string, error) {
	dir, err := os.MkdirTemp("", pattern)
	if err != nil {
		return "", err
	}

	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		return "", err
	}
	return dir, nil
}

func getOrderedKeys(m map[string]compdesc.ComponentReference) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
