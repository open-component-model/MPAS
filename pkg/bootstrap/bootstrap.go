// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package bootstrap

type Boootstrap struct {
	// The owner of the management repository
	Owner string `json:"owner,omitempty"`
	// The name of the management repository
	Repository string `json:"repository,omitempty"`
	// The public key to use for the management repository
	PublicKey string `json:"publicKey,omitempty"`
	// The path to a file containing the bootstrap component in archive format
	FromFile string `json:"fromFile,omitempty"`
	// The registry to use for the management repository
	Registry string `json:"registry,omitempty"`
}

func New() *Boootstrap {
	return &Boootstrap{}
}

// getOCMComponent returns the bootstrap component in archive format
func (b *Boootstrap) getOCMComponent() ([]byte, error) {
	return nil, nil
}

func (b *Boootstrap) unarchiveOCMComponent() error {
	return nil
}

func (b *Boootstrap) installComponent() error {
	return nil
}

func (b *Boootstrap) createManagementRepository() error {
	return nil
}
