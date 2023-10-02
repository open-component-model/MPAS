// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package create

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fluxcd/pkg/apis/meta"
	"github.com/open-component-model/mpas/cmd/mpas/config"
	"github.com/open-component-model/mpas/internal/kubeutils"
	"github.com/open-component-model/mpas/internal/printer"
	"github.com/open-component-model/mpas/internal/resource"
	rep1alpha1 "github.com/open-component-model/replication-controller/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ComponentSubscriptionCmd defines the command for creating a component subscription.
type ComponentSubscriptionCmd struct {
	name string
	config.ComponentSubscriptionConfig
}

// NewComponentSubscriptionCmd returns a new command for creating a component subscription.
func NewComponentSubscriptionCmd(name string, c config.ComponentSubscriptionConfig) *ComponentSubscriptionCmd {
	return &ComponentSubscriptionCmd{
		name:                        name,
		ComponentSubscriptionConfig: c,
	}
}

// Execute executes the command and returns an error if one occurred.
func (c *ComponentSubscriptionCmd) Execute(ctx context.Context, cfg *config.MpasConfig) error {
	t, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	err = c.validate()
	if err != nil {
		return err
	}

	interval, err := time.ParseDuration(c.Interval)
	if err != nil {
		return fmt.Errorf("interval must be specified")
	}

	csub := &resource.ComponentSubscription{
		ComponentSubscription: rep1alpha1.ComponentSubscription{
			ObjectMeta: metav1.ObjectMeta{
				Name:      c.name,
				Namespace: *cfg.KubeConfigArgs.Namespace,
			},
			Spec: rep1alpha1.ComponentSubscriptionSpec{
				Component: c.Component,
				Interval:  metav1.Duration{Duration: interval},
				Source: rep1alpha1.OCMRepository{
					URL: c.SourceUrl,
				},
			},
		},
	}

	if c.SourceSecretRef != "" {
		csub.Spec.Source.SecretRef = &meta.LocalObjectReference{
			Name: c.SourceSecretRef,
		}
	}

	if c.DestinationUrl != "" {
		csub.Spec.Destination = &rep1alpha1.OCMRepository{
			URL: c.DestinationUrl,
			SecretRef: &meta.LocalObjectReference{
				Name: c.DestinationSecretRef,
			},
		}
	}

	if c.Semver != "" {
		csub.Spec.Semver = c.Semver
	}

	if c.Verify != nil {
		signatures := make([]rep1alpha1.Signature, len(c.Verify))
		for _, v := range c.Verify {
			name, pubkey := strings.Split(v, ":")[0], strings.Split(v, ":")[1]
			signatures = append(signatures, rep1alpha1.Signature{
				Name: name,
				PublicKey: rep1alpha1.SecretRef{
					SecretRef: meta.LocalObjectReference{
						Name: pubkey,
					},
				},
			})
		}
		csub.Spec.Verify = signatures
	}

	if c.ServiceAccount != "" {
		csub.Spec.ServiceAccountName = c.ServiceAccount
	}

	if cfg.Export {
		exp, err := csub.ToYamlExport()
		if err != nil {
			return fmt.Errorf("failed to export component subscription: %w", err)
		}
		cfg.Printer.Println(exp)
		return nil
	}

	kubeClient, err := kubeutils.KubeClient(cfg.KubeConfigArgs)
	if err != nil {
		return err
	}

	if err := cfg.Printer.PrintSpinner(fmt.Sprintf("Creating component subscription %s in namespace %s",
		printer.BoldBlue(csub.Name), printer.BoldBlue(csub.Namespace))); err != nil {
		return err
	}

	if _, err := resource.ApplyAndWait(ctx, kubeClient, csub, cfg.PollInterval, t, func() error {
		return nil
	}); err != nil {
		if er := cfg.Printer.StopFailSpinner(fmt.Sprintf("Creating component subscription %s in namespace %s",
			printer.BoldBlue(csub.Name), printer.BoldBlue(csub.Namespace))); er != nil {
			err = errors.Join(err, er)
		}
		return err
	}

	return nil
}

func (c *ComponentSubscriptionCmd) validate() error {
	if c.Component == "" {
		return fmt.Errorf("component must be specified")
	}

	if c.SourceUrl == "" {
		return fmt.Errorf("source-url must be specified")
	}

	return nil
}
