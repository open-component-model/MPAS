// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package printer

import (
	"fmt"
	"io"

	"github.com/fatih/color"
)

// Printer is a wrapper around the fmt package to print to a defined output.
type Printer struct {
	output io.Writer
}

// Newprinter returns a new Printer.
func Newprinter(format string, output io.Writer) *Printer {
	return &Printer{
		output: output,
	}
}

// Printf is a convenience method to Printf to the defined output.
func (p *Printer) Printf(format string, i ...interface{}) {
	fmt.Fprintf(p.out(), format, i...)
}

// Println is a convenience method to Println to the defined output.
func (p *Printer) Println(i ...interface{}) {
	fmt.Fprintln(p.out(), i...)
}

// Print is a convenience method to Print to the defined output.
func (p *Printer) Print(i ...interface{}) {
	fmt.Fprint(p.out(), i...)
}

// out returns the output to use.
func (p *Printer) out() io.Writer {
	if p.output != nil {
		return p.output
	}
	return io.Discard
}

// BoldBlue returns a string formatted with blue and bold.
func BoldBlue(msg interface{}) string {
	return color.New(color.FgBlue).Add(color.Bold).Sprint(msg)
}

// BoldRed returns a string formatted with red and bold.
func BoldRed(msg interface{}) string {
	return color.New(color.FgRed).Add(color.Bold).Sprint(msg)
}
