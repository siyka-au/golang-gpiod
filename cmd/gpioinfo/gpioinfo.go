// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

//go:build linux
// +build linux

// A clone of libgpiocdev gpioinfo.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/warthog618/config"
	"github.com/warthog618/config/pflag"
	"github.com/warthog618/go-gpiocdev"
)

var version = "undefined"

func main() {
	flags := loadConfig()
	rc := 0
	cc := append([]string(nil), flags.Args()...)
	if len(cc) == 0 {
		cc = gpiocdev.Chips()
	}
	for _, path := range cc {
		c, err := gpiocdev.NewChip(path)
		if err != nil {
			logErr(err)
			rc = 1
			continue
		}
		fmt.Printf("%s - %d lines:\n", c.Name, c.Lines())
		for o := 0; o < c.Lines(); o++ {
			li, err := c.LineInfo(o)
			if err != nil {
				logErr(err)
				rc = 1
				continue
			}
			printLineInfo(li)
		}
		c.Close()
	}
	os.Exit(rc)
}

func loadConfig() *pflag.Getter {
	ff := []pflag.Flag{
		{Short: 'h', Name: "help", Options: pflag.IsBool},
		{Short: 'v', Name: "version", Options: pflag.IsBool},
	}
	flags := pflag.New(pflag.WithFlags(ff))
	cfg := config.New(flags)
	if v, _ := cfg.Get("help"); v.Bool() {
		printHelp()
		os.Exit(0)
	}
	if v, _ := cfg.Get("version"); v.Bool() {
		printVersion()
		os.Exit(0)
	}
	return flags
}

func logErr(err error) {
	fmt.Fprintln(os.Stderr, "gpioinfo:", err)
}

func printLineInfo(li gpiocdev.LineInfo) {
	if len(li.Name) == 0 {
		li.Name = "unnamed"
	}
	if li.Used {
		if len(li.Consumer) == 0 {
			li.Consumer = "kernel"
		}
		if strings.Contains(li.Consumer, " ") {
			li.Consumer = "\"" + li.Consumer + "\""
		}
	} else {
		li.Consumer = "unused"
	}
	dirn := "input"
	if li.Config.Direction == gpiocdev.LineDirectionOutput {
		dirn = "output"
	}
	active := "active-high"
	if li.Config.ActiveLow {
		active = "active-low"
	}
	attrs := []string(nil)
	if li.Used {
		attrs = append(attrs, "used")
	}
	switch li.Config.Drive {
	case gpiocdev.LineDriveOpenDrain:
		attrs = append(attrs, "open-drain")
	case gpiocdev.LineDriveOpenSource:
		attrs = append(attrs, "open-source")
	}
	switch li.Config.Bias {
	case gpiocdev.LineBiasPullUp:
		attrs = append(attrs, "pull-up")
	case gpiocdev.LineBiasPullDown:
		attrs = append(attrs, "pull-down")
	case gpiocdev.LineBiasDisabled:
		attrs = append(attrs, "bias-disabled")
	}
	if li.Config.DebouncePeriod != 0 {
		attrs = append(attrs,
			fmt.Sprintf("debounce-period=%s", li.Config.DebouncePeriod))
	}
	attrstr := ""
	if len(attrs) > 0 {
		attrstr = "[" + strings.Join(attrs, " ") + "]"
	}
	fmt.Printf("\tline %3d:%12s%12s%8s%13s%s\n",
		li.Offset, li.Name, li.Consumer, dirn, active, attrstr)
}

func printHelp() {
	fmt.Printf("Usage: %s [OPTIONS] <gpiochip1> ...\n", os.Args[0])
	fmt.Println("Print information about all lines of the specified GPIO chip(s) (or all gpiochips if none are specified).")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  -h, --help:\t\tdisplay this message and exit")
	fmt.Println("  -v, --version:\tdisplay the version and exit")
}

func printVersion() {
	fmt.Printf("%s (gpiocdev) %s\n", os.Args[0], version)
}
