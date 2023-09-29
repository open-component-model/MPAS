// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package create

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fluxcd/pkg/apis/meta"
	prd1alpha1 "github.com/open-component-model/mpas-product-controller/api/v1alpha1"
	"github.com/open-component-model/mpas/cmd/mpas/config"
	"github.com/open-component-model/mpas/internal/kubeutils"
	"github.com/open-component-model/mpas/internal/printer"
	"github.com/open-component-model/mpas/internal/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProductDeploymentGeneratorCmd defines the command for creating a product deployment generator.
type ProductDeploymentGeneratorCmd struct {
	name string
	config.ProductDeploymentGeneratorConfig
}

// NewPProductDeploymentGeneratorCmd returns a new command for creating a product deployment generator.
func NewProductDeploymentGeneratorCmd(name string, p config.ProductDeploymentGeneratorConfig) *ProductDeploymentGeneratorCmd {
	return &ProductDeploymentGeneratorCmd{
		name:                             name,
		ProductDeploymentGeneratorConfig: p,
	}
}

// Execute executes the command and returns an error if one occurred.
func (p *ProductDeploymentGeneratorCmd) Execute(ctx context.Context, cfg *config.MpasConfig) error {
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

	prd := &resource.ProductDeploymentGenerator{
		ProductDeploymentGenerator: prd1alpha1.ProductDeploymentGenerator{
			ObjectMeta: metav1.ObjectMeta{
				Name:      p.name,
				Namespace: *cfg.KubeConfigArgs.Namespace,
			},
			Spec: prd1alpha1.ProductDeploymentGeneratorSpec{
				SubscriptionRef: meta.NamespacedObjectReference{
					Name:      p.SubscriptionName,
					Namespace: p.SubscriptionNamespace,
				},
				Interval: metav1.Duration{
					Duration: interval,
				},
				ServiceAccountName: p.ServiceAccount,
			},
		},
	}

	if p.RepositoryName != "" {
		prd.Spec.RepositoryRef = &meta.LocalObjectReference{
			Name: p.RepositoryName,
		}
	}

	if cfg.Export {
		exp, err := prd.ToYamlExport()
		if err != nil {
			return fmt.Errorf("failed to export product deployment generator: %w", err)
		}
		cfg.Printer.Println(exp)
		return nil
	}

	kubeClient, err := kubeutils.KubeClient(cfg.KubeConfigArgs)
	if err != nil {
		return err
	}

	if err := cfg.Printer.PrintSpinner(fmt.Sprintf("Creating product deployment generator %s in namespace %s",
		printer.BoldBlue(prd.Name), printer.BoldBlue(prd.Namespace))); err != nil {
		return err
	}

	if _, err := resource.ApplyAndWait(ctx, kubeClient, prd, cfg.PollInterval, t, func() error {
		return nil
	}); err != nil {
		if er := cfg.Printer.StopFailSpinner(fmt.Sprintf("Creating product deployment generator %s in namespace %s",
			printer.BoldBlue(prd.Name), printer.BoldBlue(prd.Namespace))); er != nil {
			err = errors.Join(err, er)
		}
		return err
	}

	return nil
}

func (p *ProductDeploymentGeneratorCmd) validate() error {
	if p.SubscriptionName == "" || p.SubscriptionNamespace == "" {
		return fmt.Errorf("subscription-name must be specified")
	}

	if p.ServiceAccount == "" {
		return fmt.Errorf("service-account must be specified")
	}

	return nil
}
