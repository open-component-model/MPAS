// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package create

import (
	"context"
	"errors"
	"fmt"
	"time"

	gcv1alpha1 "github.com/open-component-model/git-controller/apis/mpas/v1alpha1"
	prj1alpha1 "github.com/open-component-model/mpas-project-controller/api/v1alpha1"
	"github.com/open-component-model/mpas/cmd/mpas/config"
	"github.com/open-component-model/mpas/internal/kubeutils"
	"github.com/open-component-model/mpas/internal/printer"
	"github.com/open-component-model/mpas/internal/resource"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProjectCmd defines the command for creating a project.
type ProjectCmd struct {
	name string
	config.ProjectConfig
}

// NewProjectCmd returns a new command for creating a project.
func NewProjectCmd(name string, p config.ProjectConfig) *ProjectCmd {
	return &ProjectCmd{
		name:          name,
		ProjectConfig: p,
	}
}

// Execute executes the command and returns an error if one occurred.
func (p *ProjectCmd) Execute(ctx context.Context, cfg *config.MpasConfig) error {
	t, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	err = p.validate()
	if err != nil {
		return err
	}

	interval, err := time.ParseDuration(p.Interval)
	if err != nil {
		return fmt.Errorf("interval must be specified")
	}

	project := &resource.Project{
		Project: prj1alpha1.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name:      p.name,
				Namespace: *cfg.KubeConfigArgs.Namespace,
			},
			Spec: prj1alpha1.ProjectSpec{
				Git: gcv1alpha1.RepositorySpec{
					Owner:    p.Owner,
					Provider: p.Provider,
					Credentials: gcv1alpha1.Credentials{
						SecretRef: v1.LocalObjectReference{
							Name: p.SecretRef,
						},
					},
					DefaultBranch: p.Branch,
					Interval: metav1.Duration{
						Duration: interval,
					},
					Visibility:               p.Visibility,
					IsOrganization:           !p.Personal,
					Domain:                   p.Domain,
					Maintainers:              p.Maintainers,
					ExistingRepositoryPolicy: gcv1alpha1.ExistingRepositoryPolicy(p.AlreadyExistsPolicy),
				},
				Flux: prj1alpha1.FluxSpec{
					Interval: metav1.Duration{
						Duration: interval,
					},
				},
				Prune: p.Prune,
				Interval: metav1.Duration{
					Duration: interval,
				},
			},
		},
	}

	if p.Email != "" || p.Message != "" || p.Author != "" {
		project.Spec.Git.CommitTemplate = &gcv1alpha1.CommitTemplate{
			Email:   p.Email,
			Message: p.Message,
			Name:    p.Author,
		}
	}

	if cfg.Export {
		exp, err := project.ToYamlExport()
		if err != nil {
			return fmt.Errorf("failed to export project: %w", err)
		}
		cfg.Printer.Println(exp)
		return nil
	}

	kubeClient, err := kubeutils.KubeClient(cfg.KubeConfigArgs)
	if err != nil {
		return err
	}

	if err := cfg.Printer.PrintSpinner(fmt.Sprintf("Creating project %s in namespace %s",
		printer.BoldBlue(project.Name), printer.BoldBlue(project.Namespace))); err != nil {
		return err
	}

	if _, err := resource.ApplyAndWait(ctx, kubeClient, project, cfg.PollInterval, t, func() error {
		return nil
	}); err != nil {
		if er := cfg.Printer.StopFailSpinner(fmt.Sprintf("Creating project %s in namespace %s",
			printer.BoldBlue(project.Name), printer.BoldBlue(project.Namespace))); er != nil {
			err = errors.Join(err, er)
		}
		return err
	}

	return nil
}

func (p *ProjectCmd) validate() error {
	if p.Owner == "" {
		return fmt.Errorf("owner must be specified")
	}

	if p.Provider == "" {
		return fmt.Errorf("provider must be specified")
	}

	if p.SecretRef == "" {
		return fmt.Errorf("secret-ref must be specified")
	}
	return nil
}
