// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/open-component-model/mpas/internal/env"
	"github.com/open-component-model/mpas/internal/kubeutils"
	"github.com/open-component-model/mpas/internal/ocm"
	"github.com/open-component-model/mpas/internal/printer"
	om "github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/utils"
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
	caFile                string
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

func WithRootFile(caFile string) Option {
	return func(o *options) {
		o.caFile = caFile
	}
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
	octx := om.DefaultContext()
	if _, err := utils.Configure(octx, ""); err != nil {
		return fmt.Errorf("failed to configure ocm context: %w", err)
	}
	// set default log level to 1 which is ERROR level to avoid printing INFO messages
	octx.LoggingContext().SetDefaultLevel(1)

	b.printer.Printf("Running %s ...\n",
		printer.BoldBlue("mpas bootstrap"))

	if err := b.inSpinner(fmt.Sprintf("Preparing Management repository %s",
		printer.BoldBlue(b.repositoryName)), func() error {
		return b.reconcileManagementRepository(ctx)
	}); err != nil {
		return fmt.Errorf("failed to prepare management repository: %w", err)
	}

	if b.fromFile != "" {
		fromFileToOciRepo := func() error {
			ctf, err := ocm.RepositoryFromCTF(b.fromFile)
			if err != nil {
				return fmt.Errorf("failed to create CTF from file %q: %w", b.fromFile, err)
			}
			defer ctf.Close()

			target, err := ocm.MakeRepositoryWithDockerConfig(octx, b.registry, b.dockerConfigPath)
			if err != nil {
				return fmt.Errorf("failed to create target repository: %w", err)
			}
			defer target.Close()

			if err := ocm.Transfer(octx, ctf, target, io.Discard); err != nil {
				return fmt.Errorf("failed to transfer CTF from %q to %q: %w", b.fromFile, b.registry, err)
			}

			return nil
		}

		if err := b.inSpinner(fmt.Sprintf("Transferring bootstrap component from %s to %s",
			printer.BoldBlue(b.fromFile), printer.BoldBlue(b.registry)), fromFileToOciRepo); err != nil {
			return fmt.Errorf("failed to prepare from file: %w", err)
		}
	}

	var (
		refs    map[string]compdesc.ComponentReference
		ociRepo om.Repository
		err     error
	)

	if err := b.inSpinner(fmt.Sprintf("Fetching bootstrap component from %s",
		printer.BoldBlue(b.registry)), func() error {
		ociRepo, err = ocm.MakeRepositoryWithDockerConfig(octx, b.registry, b.dockerConfigPath)
		if err != nil {
			return fmt.Errorf("failed to fetch bootstrap component references: %w", err)
		}

		refs, err = b.fetchBootstrapComponentReferences(ociRepo)
		if err != nil {
			return fmt.Errorf("failed to fetch bootstrap component references: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to fetch bootstrap components: %w", err)
	}

	sha, err := b.installInfrastructure(ctx, ociRepo, refs)
	if err != nil {
		return fmt.Errorf("failed to install infrastructure: %w", err)
	}

	if err := b.inSpinner("Reconciling infrastructure components", func() error {
		return b.syncManagementRepository(ctx, sha)
	}); err != nil {
		return err
	}

	if err := b.inSpinner("Waiting for cert-manager to be available", func() error {
		if err := kubeutils.ReportComponentsHealth(ctx, b.restClientGetter, b.timeout, []string{
			certManager,
			certManagerCAInjector,
			certManagerWebhook,
		}, "cert-manager"); err != nil {
			return fmt.Errorf("failed to report health, please try again in a few minutes: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to wait for cert-manager to be available: %w", err)
	}

	compNs := make(map[string][]string)
	var latestSHA string
	// install components in order by using the ordered keys
	comps := getOrderedKeys(refs)
	for _, comp := range comps {
		ref := refs[comp]

		if err := b.inSpinner(fmt.Sprintf("Generating %s manifest with version %s",
			printer.BoldBlue(comp),
			printer.BoldBlue(ref.GetVersion())), func() error {
			latestSHA, err = b.generateControllerManifest(ctx, ociRepo, comp, ref, compNs)
			if err != nil {
				return err
			}

			return nil
		}); err != nil {
			return fmt.Errorf("failed to generate manifest: %w", err)
		}
	}

	if err := b.inSpinner("Generate certificate manifests", func() error {
		latestSHA, err = b.generateCertificateManifests(ctx)

		if err != nil {
			return fmt.Errorf("failed to generate manifests: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to generate certificate manifests: %w", err)
	}

	if err := b.inSpinner("Reconciling infrastructure components", func() error {
		return b.syncManagementRepository(ctx, latestSHA)
	}); err != nil {
		return err
	}

	if err := b.inSpinner("Waiting for components to be ready", func() error {
		for ns, comps := range compNs {
			if err := kubeutils.ReportComponentsHealth(ctx, b.restClientGetter, b.timeout, comps, ns); err != nil {
				return fmt.Errorf("failed to report health, please try again in a few minutes: %w", err)
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to wait for components to be ready: %w", err)
	}

	b.printer.Printf("\n")
	b.printer.Printf("Bootstrap completed successfully!\n")

	return nil
}

func (b *Bootstrap) inSpinner(msg string, f func() error) (err error) {
	if err := b.printer.PrintSpinner(msg); err != nil {
		return err
	}

	if err := f(); err != nil {
		if serr := b.printer.StopFailSpinner(msg); serr != nil {
			err = errors.Join(err, serr)
		}

		return err
	}

	return b.printer.StopSpinner(msg)
}

func (b *Bootstrap) syncManagementRepository(ctx context.Context, latestSHA string) error {
	expectedRevision := fmt.Sprintf("%s@sha1:%s", b.defaultBranch, latestSHA)
	if err := kubeutils.ReconcileGitrepository(ctx, b.kubeclient, env.DefaultFluxNamespace, env.DefaultFluxNamespace); err != nil {
		return err
	}

	if err := kubeutils.ReportGitrepositoryHealth(ctx, b.kubeclient, env.DefaultFluxNamespace, env.DefaultFluxNamespace, expectedRevision, env.DefaultPollInterval, b.timeout); err != nil {
		return fmt.Errorf("failed to report gitrepository health: %w", err)
	}

	if err := kubeutils.ReconcileKustomization(ctx, b.kubeclient, env.DefaultFluxNamespace, env.DefaultFluxNamespace); err != nil {
		return err
	}

	if err := kubeutils.ReportKustomizationHealth(ctx, b.kubeclient, env.DefaultFluxNamespace, env.DefaultFluxNamespace, expectedRevision, env.DefaultPollInterval, b.timeout); err != nil {
		return fmt.Errorf("failed to report kustomization health: %w", err)
	}

	return nil
}

func (b *Bootstrap) installComponent(ctx context.Context, ociRepo om.Repository, ref compdesc.ComponentReference, comp, ns string, compNs map[string][]string) (string, error) {
	dir, err := mkdirTempDir(fmt.Sprintf("%s-install", comp))
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	opts := &componentOptions{
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

	var caBundle []byte
	if b.caFile != "" {
		caBundle, err = os.ReadFile(b.caFile)
		if err != nil {
			return fmt.Errorf("failed to read CA file: %w", err)
		}
	}

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
		caFile:                caBundle,
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

func (b *Bootstrap) installCertManager(ctx context.Context, ociRepo om.Repository, ref compdesc.ComponentReference) (string, error) {
	dir, err := mkdirTempDir("cert-manager-install")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)

	opts := &certManagerOptions{
		gitRepository:         b.repository,
		dir:                   dir,
		branch:                b.defaultBranch,
		targetPath:            b.targetPath,
		namespace:             "cert-manager",
		provider:              string(b.providerClient.ProviderID()),
		timeout:               b.timeout,
		commitMessageAppendix: b.commitMessageAppendix,
	}

	inst, err := newCertManagerInstall(ref.GetComponentName(), ref.GetVersion(), ociRepo, opts)
	if err != nil {
		return "", fmt.Errorf("failed to create new cert manager installer: %w", err)
	}
	sha, err := inst.Install(ctx, "cert-manager")
	if err != nil {
		return "", fmt.Errorf("failed to install cert manager: %w", err)
	}
	return sha, nil
}

func (b *Bootstrap) fetchBootstrapComponentReferences(ociRepo om.Repository) (map[string]compdesc.ComponentReference, error) {
	cv, err := ocm.FetchLatestComponentVersion(ociRepo, env.DefaultBootstrapComponent)
	if err != nil {
		return nil, err
	}
	defer cv.Close()

	return ocm.FetchComponentReferences(cv, b.components)
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

func (b *Bootstrap) installInfrastructure(ctx context.Context, ociRepo om.Repository, refs map[string]compdesc.ComponentReference) (string, error) {
	fluxRef, ok := refs[env.FluxName]
	if !ok {
		return "", fmt.Errorf("flux component not found")
	}

	if err := b.inSpinner(fmt.Sprintf("Installing %s with version %s",
		printer.BoldBlue(env.FluxName),
		printer.BoldBlue(fluxRef.GetVersion())), func() error {

		return b.installFlux(ctx, ociRepo, fluxRef)
	}); err != nil {
		return "", fmt.Errorf("failed to install flux: %w", err)
	}

	delete(refs, env.FluxName)

	certManagerRef, ok := refs[env.CertManagerName]
	if !ok {
		return "", fmt.Errorf("cert-manager component not found")
	}

	var (
		sha string
		err error
	)
	if err := b.inSpinner(fmt.Sprintf("Installing %s with version %s",
		printer.BoldBlue(env.CertManagerName),
		printer.BoldBlue(certManagerRef.GetVersion())), func() error {
		sha, err = b.installCertManager(ctx, ociRepo, certManagerRef)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return "", fmt.Errorf("failed to install cert-manager: %w", err)
	}

	delete(refs, env.CertManagerName)

	return sha, nil
}

func (b *Bootstrap) generateControllerManifest(ctx context.Context, ociRepo om.Repository, comp string, ref compdesc.ComponentReference, compNs map[string][]string) (string, error) {
	var latestSHA string
	switch comp {
	case env.OcmControllerName, env.GitControllerName, env.ReplicationControllerName:
		sha, err := b.installComponent(ctx, ociRepo, ref, comp, env.DefaultOCMNamespace, compNs)
		if err != nil {
			return "", err
		}
		latestSHA = sha
		compNs[env.DefaultOCMNamespace] = append(compNs[env.DefaultOCMNamespace], comp)
	case env.MpasProductControllerName, env.MpasProjectControllerName:
		sha, err := b.installComponent(ctx, ociRepo, ref, comp, "mpas-system", compNs)
		if err != nil {
			return "", err
		}
		latestSHA = sha
		compNs["mpas-system"] = append(compNs["mpas-system"], comp)
	default:
		return "", fmt.Errorf("unknown component %q", comp)
	}

	return latestSHA, nil
}

func (b *Bootstrap) generateCertificateManifests(ctx context.Context) (string, error) {
	installer := newCertificateManifestInstaller(&certificateManifestOptions{
		gitRepository:         b.repository,
		branch:                b.defaultBranch,
		targetPath:            b.targetPath,
		provider:              string(b.providerClient.ProviderID()),
		timeout:               b.timeout,
		commitMessageAppendix: b.commitMessageAppendix,
		kubeClient:            b.kubeclient,
	})

	return installer.Install(ctx)
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
