// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

// An example of reading values from a MCP3008 using the bit-bashed SPI driver.
package main

import (
	"fmt"
	"os"

	"github.com/warthog618/config"
	"github.com/warthog618/config/blob"
	"github.com/warthog618/config/blob/decoder/json"
	"github.com/warthog618/config/dict"
	"github.com/warthog618/config/env"
	"github.com/warthog618/config/pflag"
	"github.com/warthog618/go-gpiocdev"
	"github.com/warthog618/go-gpiocdev/device/rpi"
	"github.com/warthog618/go-gpiocdev/spi/mcp3w0c"
)

// This example reads both channels from an MCP3008 connected to the RPI by four
// data lines - CSZ, CLK, DI, and DO. The default pin assignments are defined in
// loadConfig, but can be altered via configuration (env, flag or config file).
// All pins other than DO are outputs so do not run this example on a board
// where those pins serve other purposes.
func main() {
	cfg := loadConfig()
	tclk := cfg.MustGet("tclk").Duration()
	tset := cfg.MustGet("tset").Duration()
	if tset < tclk {
		tset = 0
	} else {
		tset -= tclk
	}
	chip := cfg.MustGet("gpiochip").String()
	c, err := gpiocdev.NewChip(chip, gpiocdev.WithConsumer("mcp3008"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "mcp3008: %s\n", err)
		os.Exit(1)
	}
	adc, err := mcp3w0c.NewMCP3008(
		c,
		cfg.MustGet("clk").Int(),
		cfg.MustGet("csz").Int(),
		cfg.MustGet("di").Int(),
		cfg.MustGet("do").Int(),
		mcp3w0c.WithTclk(tclk),
		mcp3w0c.WithTset(tset),
	)
	c.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "mcp3008: %s\n", err)
		os.Exit(1)
	}
	defer adc.Close()
	for ch := 0; ch < 8; ch++ {
		d, err := adc.Read(ch)
		if err != nil {
			fmt.Printf("error reading ch%d: %s\n", ch, err)
			continue
		}
		fmt.Printf("ch%d=0x%04x\n", ch, d)
	}
}

func loadConfig() *config.Config {
	defaultConfig := map[string]interface{}{
		"gpiochip": "gpiochip0",
		"tclk":     "500ns",
		"tset":     "500ns",
		"clk":      rpi.J8p36,
		"csz":      rpi.J8p37,
		"di":       rpi.J8p38,
		"do":       rpi.J8p40,
	}
	def := dict.New(dict.WithMap(defaultConfig))
	flags := []pflag.Flag{
		{Short: 'c', Name: "config-file"},
	}
	// highest priority sources first - flags override environment
	cfg := config.New(
		pflag.New(pflag.WithFlags(flags)),
		env.New(env.WithEnvPrefix("MCP3008_")),
		config.WithDefault(def))
	cfg.Append(
		blob.NewConfigFile(cfg, "config.file", "mcp3008.json", json.NewDecoder()))
	cfg = cfg.GetConfig("", config.WithMust())
	return cfg
}
