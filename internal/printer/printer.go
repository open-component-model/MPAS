// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package printer

import (
	"fmt"
	"io"
	"time"

	"github.com/fatih/color"
	"github.com/theckman/yacspin"
)

// Printer is a wrapper around the fmt package to print to a defined output.
type Printer struct {
	output  io.Writer
	spinner *yacspin.Spinner
}

// Newprinter returns a new Printer.
func Newprinter(output io.Writer) (*Printer, error) {
	cfg := yacspin.Config{
		Frequency:         200 * time.Millisecond,
		CharSet:           yacspin.CharSets[26],
		Prefix:            " ",
		Suffix:            " ",
		SuffixAutoColon:   true,
		StopCharacter:     "✓",
		StopColors:        []string{"fgGreen"},
		StopFailCharacter: "✗",
		StopFailColors:    []string{"fgRed"},
	}
	spinner, err := yacspin.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create spinner: %w", err)
	}
	return &Printer{
		output:  output,
		spinner: spinner,
	}, nil
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

func (p *Printer) SetOutput(output io.Writer) {
	p.output = output
}

func (p *Printer) startSpinner() error {
	if p.spinner.Status() == yacspin.SpinnerStopped {
		err := p.spinner.Start()
		if err != nil {
			return fmt.Errorf("failed to start spinner: %w", err)
		}
	}
	return nil
}

// PrintSpinner starts a spinner and returns a function to stop it.
func (p *Printer) PrintSpinner(message string) error {
	p.spinner.Message(message)
	err := p.startSpinner()
	if err != nil {
		return err
	}
	return nil
}

func (p *Printer) StopSpinner(message string) error {
	p.spinner.StopMessage(message)
	err := p.spinner.Stop()
	if err != nil {
		return fmt.Errorf("failed to stop spinner: %w", err)
	}
	return nil
}

func (p *Printer) StopFailSpinner(message string) error {
	p.spinner.StopFailMessage(message)
	err := p.spinner.StopFail()
	if err != nil {
		return fmt.Errorf("failed to stop spinner: %w", err)
	}
	return nil
}

// BoldBlue returns a string formatted with blue and bold.
func BoldBlue(msg interface{}) string {
	return color.New(color.FgBlue).Add(color.Bold).Sprint(msg)
}

// BoldRed returns a string formatted with red and bold.
func BoldRed(msg interface{}) string {
	return color.New(color.FgRed).Add(color.Bold).Sprint(msg)
}
