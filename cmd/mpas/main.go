// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
)

func main() {
	config := MpasConfig{}
	cmd := New(config, os.Args[1:])
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
