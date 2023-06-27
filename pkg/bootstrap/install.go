// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import "context"

type Installer interface {
	Install(ctx context.Context, component string) error
	Cleanup(ctx context.Context) error
}
