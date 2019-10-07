// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// Package mcp3w0c provides bit bashed device drivers for MCP3004/3008/3204/3208
// SPI ADCs.
package mcp3w0c

import (
	"errors"
	"sync"
	"time"

	"github.com/warthog618/gpiod"
	"github.com/warthog618/gpiod/spi"
)

// MCP3w0c reads ADC values from a connected Microchip MCP3xxx family device.
//
// Supported variants are MCP3004/3008/3204/3208.
// The w indicates the width of the device (0 => 10, 2 => 12)
// and the c the number of channels.
type MCP3w0c struct {
	mu    sync.Mutex
	s     *spi.SPI
	width uint
	// time to allow mux to settle after clocking out channel
	tset time.Duration
}

// New creates a MCP3w0c.
func New(c *gpiod.Chip, clk, csz, di, do int, width uint, options ...Option) (*MCP3w0c, error) {
	s, err := spi.New(c, clk, csz, di, do, spi.WithTclk(500*time.Nanosecond))
	if err != nil {
		return nil, err
	}
	a := MCP3w0c{s: s, width: width}
	for _, option := range options {
		option(&a)
	}
	return &a, nil
}

// NewMCP3008 creates a MCP3008.
func NewMCP3008(c *gpiod.Chip, clk, csz, di, do int, options ...Option) (*MCP3w0c, error) {
	return New(c, clk, csz, di, do, 10, options...)
}

// NewMCP3208 creates a MCP3208.
func NewMCP3208(c *gpiod.Chip, clk, csz, di, do int, options ...Option) (*MCP3w0c, error) {
	return New(c, clk, csz, di, do, 12, options...)
}

// Close releases all resources allocated to the ADC.
func (adc *MCP3w0c) Close() error {
	adc.mu.Lock()
	defer adc.mu.Unlock()
	if adc.s == nil {
		return ErrClosed
	}
	adc.s.Close()
	adc.s = nil
	return nil
}

// Read returns the value of a single channel read from the ADC.
func (adc *MCP3w0c) Read(ch int) (uint16, error) {
	return adc.read(ch, 1)
}

// ReadDifferential returns the value of a differential pair read from the ADC.
func (adc *MCP3w0c) ReadDifferential(ch int) (uint16, error) {
	return adc.read(ch, 0)
}

// ErrClosed indicates the ADC is closed.
var ErrClosed = errors.New("closed")

func (adc *MCP3w0c) read(ch int, sgl int) (uint16, error) {
	adc.mu.Lock()
	defer adc.mu.Unlock()
	if adc.s == nil {
		return 0, ErrClosed
	}
	s := adc.s
	err := s.Ssz.SetValue(1)
	if err != nil {
		return 0, err
	}
	err = s.Sclk.SetValue(0)
	if err != nil {
		return 0, err
	}
	if s.Mosi == s.Miso {
		panic("setting line direction not currently supported")
		// !!! s.Mosi.Output(1)
	}
	err = s.Mosi.SetValue(1)
	if err != nil {
		return 0, err
	}
	time.Sleep(s.Tclk)
	err = s.Ssz.SetValue(0)
	if err != nil {
		return 0, err
	}

	err = s.ClockOut(1) // Start
	if err != nil {
		return 0, err
	}
	err = s.ClockOut(sgl) // SGL/DIFFZ
	if err != nil {
		return 0, err
	}
	for i := 2; i >= 0; i-- {
		d := 0
		if (ch >> uint(i) & 0x01) == 0x01 {
			d = 1
		}
		err = s.ClockOut(d)
		if err != nil {
			return 0, err
		}
	}
	// mux settling
	if s.Mosi == s.Miso {
		panic("setting line direction not currently supported")
		// !!! s.Miso.Input()
	}
	time.Sleep(adc.tset)
	_, err = s.ClockIn() // sample time - junk
	if err != nil {
		return 0, err
	}

	_, err = s.ClockIn() // null bit
	if err != nil {
		return 0, err
	}

	var d uint16
	for i := uint(0); i < adc.width; i++ {
		v, err := s.ClockIn()
		if err != nil {
			return 0, err
		}
		d = d << 1
		if v != 0 {
			d = d | 0x01
		}
	}
	err = s.Ssz.SetValue(1)
	if err != nil {
		return 0, err
	}
	return d, nil
}

// Option specifies a construction option for the ADC.
type Option func(*MCP3w0c)

// WithTclk sets the clock period for the ADC.
//
// Note that this is the half-cycle period.
func WithTclk(tclk time.Duration) Option {
	return func(a *MCP3w0c) {
		a.s.Tclk = tclk
	}
}

// WithTset sets the settling period for the ADC.
func WithTset(tset time.Duration) Option {
	return func(a *MCP3w0c) {
		a.tset = tset
	}
}
