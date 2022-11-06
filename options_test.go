// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

package gpiocdev_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/warthog618/go-gpiocdev"
	"github.com/warthog618/go-gpiocdev/mockup"
	"golang.org/x/sys/unix"
)

func TestWithConsumer(t *testing.T) {
	// default from chip
	c := getChip(t, gpiocdev.WithConsumer("gpiocdev-test-chip"))
	defer c.Close()
	l, err := c.RequestLine(platform.IntrLine())
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err := c.LineInfo(platform.IntrLine())
	assert.Nil(t, err)
	assert.Equal(t, "gpiocdev-test-chip", inf.Consumer)
	err = l.Close()
	assert.Nil(t, err)

	// overridden by line
	l, err = c.RequestLine(platform.IntrLine(),
		gpiocdev.WithConsumer("gpiocdev-test-line"))
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	inf, err = c.LineInfo(platform.IntrLine())
	assert.Nil(t, err)
	assert.Equal(t, "gpiocdev-test-line", inf.Consumer)
}

func TestAsIs(t *testing.T) {
	if !platform.SupportsAsIs() {
		t.Skip("platform doesn't support as-is")
	}
	c := getChip(t)
	defer c.Close()

	// leave input as input
	l, err := c.RequestLine(platform.FloatingLines()[0], gpiocdev.AsInput)
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err := c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.Equal(t, gpiocdev.LineDirectionInput, inf.Config.Direction)
	l.Close()
	l, err = c.RequestLine(platform.FloatingLines()[0], gpiocdev.AsIs)
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.Equal(t, gpiocdev.LineDirectionInput, inf.Config.Direction)
	err = l.Close()
	assert.Nil(t, err)

	// leave output as output
	l, err = c.RequestLine(platform.FloatingLines()[0], gpiocdev.AsOutput())
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.Equal(t, gpiocdev.LineDirectionOutput, inf.Config.Direction)
	l.Close()
	l, err = c.RequestLine(platform.FloatingLines()[0], gpiocdev.AsIs)
	assert.Nil(t, err)
	require.NotNil(t, l)
	err = l.Close()
	assert.Nil(t, err)
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.Equal(t, gpiocdev.LineDirectionOutput, inf.Config.Direction)
}

func testLineDirectionOption(t *testing.T,
	contraOption, option gpiocdev.LineReqOption, config gpiocdev.LineConfig) {

	t.Helper()

	c := getChip(t)
	defer c.Close()

	// change direction
	l, err := c.RequestLine(platform.FloatingLines()[0], contraOption)
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err := c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.NotEqual(t, config.Direction, inf.Config.Direction)
	l.Close()
	l, err = c.RequestLine(platform.FloatingLines()[0], option)
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.Equal(t, config.Direction, inf.Config.Direction)
	err = l.Close()
	assert.Nil(t, err)

	// same direction
	l, err = c.RequestLine(platform.FloatingLines()[0], option)
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.Equal(t, config.Direction, inf.Config.Direction)
	err = l.Close()
	assert.Nil(t, err)
}

func testLineDirectionReconfigure(t *testing.T, createOption gpiocdev.LineReqOption,
	reconfigOption gpiocdev.LineConfigOption, config gpiocdev.LineConfig) {

	tf := func(t *testing.T) {
		requireKernel(t, setConfigKernel)

		c := getChip(t)
		defer c.Close()
		// reconfigure direction change
		l, err := c.RequestLine(platform.FloatingLines()[0], createOption)
		assert.Nil(t, err)
		require.NotNil(t, l)
		inf, err := c.LineInfo(platform.FloatingLines()[0])
		assert.Nil(t, err)
		assert.NotEqual(t, config.Direction, inf.Config.Direction)
		l.Reconfigure(reconfigOption)
		inf, err = c.LineInfo(platform.FloatingLines()[0])
		assert.Nil(t, err)
		assert.Equal(t, config.Direction, inf.Config.Direction)
		err = l.Close()
		assert.Nil(t, err)
	}
	t.Run("Reconfigure", tf)
}

func TestAsInput(t *testing.T) {
	config := gpiocdev.LineConfig{
		Direction: gpiocdev.LineDirectionInput,
	}
	testChipAsInputOption(t)
	testLineDirectionOption(t, gpiocdev.AsOutput(), gpiocdev.AsInput, config)
	testLineDirectionReconfigure(t, gpiocdev.AsOutput(), gpiocdev.AsInput, config)
}

func TestAsOutput(t *testing.T) {
	config := gpiocdev.LineConfig{
		Direction: gpiocdev.LineDirectionOutput,
	}
	testLineDirectionOption(t, gpiocdev.AsInput, gpiocdev.AsOutput(), config)
	testLineDirectionReconfigure(t, gpiocdev.AsInput, gpiocdev.AsOutput(), config)
}

func testEdgeEventPolarity(t *testing.T, l *gpiocdev.Line,
	ich <-chan gpiocdev.LineEvent, activeLevel int, seqno uint32) {

	t.Helper()

	evtSeqno = seqno
	platform.TriggerIntr(activeLevel ^ 1)
	waitEvent(t, ich, nextEvent(l, 0))
	v, err := l.Value()
	assert.Nil(t, err)
	assert.Equal(t, 0, v)
	platform.TriggerIntr(activeLevel)
	waitEvent(t, ich, nextEvent(l, 1))
	v, err = l.Value()
	assert.Nil(t, err)
	assert.Equal(t, 1, v)
	err = l.Close()
	assert.Nil(t, err)
}

func testChipAsInputOption(t *testing.T) {
	t.Helper()

	c := getChip(t, gpiocdev.AsInput)
	defer c.Close()

	// force line to output
	l, err := c.RequestLine(platform.OutLine(),
		gpiocdev.AsOutput(0))
	assert.Nil(t, err)
	require.NotNil(t, l)
	l.Close()

	// request with Chip default
	l, err = c.RequestLine(platform.OutLine())
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	inf, err := c.LineInfo(platform.OutLine())
	assert.Nil(t, err)
	assert.Equal(t, gpiocdev.LineDirectionInput, inf.Config.Direction)
}

func testChipLevelOption(t *testing.T, option gpiocdev.ChipOption,
	isActiveLow bool, activeLevel int) {

	t.Helper()

	c := getChip(t, option)
	defer c.Close()

	platform.TriggerIntr(activeLevel)
	ich := make(chan gpiocdev.LineEvent, 3)
	l, err := c.RequestLine(platform.IntrLine(),
		gpiocdev.WithBothEdges,
		gpiocdev.WithEventHandler(func(evt gpiocdev.LineEvent) {
			ich <- evt
		}))
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	inf, err := c.LineInfo(platform.IntrLine())
	assert.Nil(t, err)
	assert.Equal(t, isActiveLow, inf.Config.ActiveLow)

	// can get initial state events on some platforms (e.g. RPi AsActiveHigh)
	seqno := clearEvents(ich)

	// test correct edge polarity in events
	testEdgeEventPolarity(t, l, ich, activeLevel, seqno)
}

func testLineLevelOptionInput(t *testing.T, option gpiocdev.LineReqOption,
	isActiveLow bool, activeLevel int) {

	t.Helper()

	c := getChip(t)
	defer c.Close()

	platform.TriggerIntr(activeLevel)
	ich := make(chan gpiocdev.LineEvent, 3)
	l, err := c.RequestLine(platform.IntrLine(),
		option,
		gpiocdev.WithBothEdges,
		gpiocdev.WithEventHandler(func(evt gpiocdev.LineEvent) {
			ich <- evt
		}))
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	inf, err := c.LineInfo(platform.IntrLine())
	assert.Nil(t, err)
	assert.Equal(t, isActiveLow, inf.Config.ActiveLow)

	testEdgeEventPolarity(t, l, ich, activeLevel, 0)
}

func testLineLevelOptionOutput(t *testing.T, option gpiocdev.LineReqOption,
	isActiveLow bool, activeLevel int) {

	t.Helper()

	c := getChip(t)
	defer c.Close()

	l, err := c.RequestLine(platform.OutLine(), option, gpiocdev.AsOutput(1))
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err := c.LineInfo(platform.OutLine())
	assert.Nil(t, err)
	assert.Equal(t, isActiveLow, inf.Config.ActiveLow)
	v := platform.ReadOut()
	assert.Equal(t, activeLevel, v)
	err = l.SetValue(0)
	assert.Nil(t, err)
	v = platform.ReadOut()
	assert.Equal(t, activeLevel^1, v)
	err = l.SetValue(1)
	assert.Nil(t, err)
	v = platform.ReadOut()
	assert.Equal(t, activeLevel, v)
	err = l.Close()
	assert.Nil(t, err)
}

func testLineLevelReconfigure(t *testing.T, createOption gpiocdev.LineReqOption,
	reconfigOption gpiocdev.LineConfigOption, isActiveLow bool, activeLevel int) {

	tf := func(t *testing.T) {
		requireKernel(t, setConfigKernel)

		c := getChip(t)
		defer c.Close()

		l, err := c.RequestLine(platform.OutLine(), createOption, gpiocdev.AsOutput(1))
		assert.Nil(t, err)
		require.NotNil(t, l)
		v := platform.ReadOut()
		assert.Equal(t, activeLevel^1, v)
		inf, err := c.LineInfo(platform.OutLine())
		assert.Nil(t, err)
		assert.NotEqual(t, isActiveLow, inf.Config.ActiveLow)
		l.Reconfigure(reconfigOption)
		inf, err = c.LineInfo(platform.OutLine())
		assert.Nil(t, err)
		assert.Equal(t, isActiveLow, inf.Config.ActiveLow)
		v = platform.ReadOut()
		assert.Equal(t, activeLevel, v)
		err = l.Close()
		assert.Nil(t, err)
	}
	t.Run("Reconfigure", tf)
}

func TestAsActiveLow(t *testing.T) {
	testChipLevelOption(t, gpiocdev.AsActiveLow, true, 0)
	testLineLevelOptionInput(t, gpiocdev.AsActiveLow, true, 0)
	testLineLevelOptionOutput(t, gpiocdev.AsActiveLow, true, 0)
	testLineLevelReconfigure(t, gpiocdev.AsActiveHigh, gpiocdev.AsActiveLow, true, 0)
}

func TestAsActiveHigh(t *testing.T) {
	testChipLevelOption(t, gpiocdev.AsActiveHigh, false, 1)
	testLineLevelOptionInput(t, gpiocdev.AsActiveHigh, false, 1)
	testLineLevelOptionOutput(t, gpiocdev.AsActiveHigh, false, 1)
	testLineLevelReconfigure(t, gpiocdev.AsActiveLow, gpiocdev.AsActiveHigh, false, 1)
}

func testChipDriveOption(t *testing.T, option gpiocdev.ChipOption,
	drive gpiocdev.LineDrive, values ...int) {

	t.Helper()

	c := getChip(t, option)
	defer c.Close()

	l, err := c.RequestLine(platform.OutLine(),
		gpiocdev.AsOutput(1))
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	inf, err := c.LineInfo(platform.OutLine())
	assert.Nil(t, err)
	assert.Equal(t, drive, inf.Config.Drive)
	for _, sv := range values {
		err = l.SetValue(sv)
		assert.Nil(t, err)
		v := platform.ReadOut()
		assert.Equal(t, sv, v)
	}
}

func testLineDriveOption(t *testing.T, option gpiocdev.LineReqOption,
	drive gpiocdev.LineDrive, values ...int) {

	t.Helper()

	c := getChip(t)
	defer c.Close()

	l, err := c.RequestLine(platform.OutLine(),
		gpiocdev.AsOutput(1), option)
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	inf, err := c.LineInfo(platform.OutLine())
	assert.Nil(t, err)
	assert.Equal(t, drive, inf.Config.Drive)
	for _, sv := range values {
		err = l.SetValue(sv)
		assert.Nil(t, err)
		v := platform.ReadOut()
		assert.Equal(t, sv, v)
	}
}

func testLineDriveReconfigure(t *testing.T, createOption gpiocdev.LineReqOption,
	reconfigOption gpiocdev.LineConfigOption, drive gpiocdev.LineDrive, values ...int) {

	tf := func(t *testing.T) {
		requireKernel(t, setConfigKernel)

		c := getChip(t)
		defer c.Close()

		l, err := c.RequestLine(platform.OutLine(),
			createOption, gpiocdev.AsOutput(1))
		assert.Nil(t, err)
		require.NotNil(t, l)
		defer l.Close()
		err = l.Reconfigure(reconfigOption)
		assert.Nil(t, err)
		inf, err := c.LineInfo(platform.OutLine())
		assert.Nil(t, err)
		assert.Equal(t, drive, inf.Config.Drive)
		for _, sv := range values {
			err = l.SetValue(sv)
			assert.Nil(t, err)
			v := platform.ReadOut()
			assert.Equal(t, sv, v)
		}
	}
	t.Run("Reconfigure", tf)
}

func TestAsOpenDrain(t *testing.T) {
	drive := gpiocdev.LineDriveOpenDrain
	// Testing float high requires specific hardware, so assume that is
	// covered by the kernel anyway...
	testChipDriveOption(t, gpiocdev.AsOpenDrain, drive, 0)
	testLineDriveOption(t, gpiocdev.AsOpenDrain, drive, 0)
	testLineDriveReconfigure(t, gpiocdev.AsOpenSource, gpiocdev.AsOpenDrain, drive, 0)
}

func TestAsOpenSource(t *testing.T) {
	drive := gpiocdev.LineDriveOpenSource
	// Testing float low requires specific hardware, so assume that is
	// covered by the kernel anyway.
	testChipDriveOption(t, gpiocdev.AsOpenSource, drive, 1)
	testLineDriveOption(t, gpiocdev.AsOpenSource, drive, 1)
	testLineDriveReconfigure(t, gpiocdev.AsOpenDrain, gpiocdev.AsOpenSource, drive, 1)
}

func TestAsPushPull(t *testing.T) {
	drive := gpiocdev.LineDrivePushPull
	testChipDriveOption(t, gpiocdev.AsPushPull, drive, 0, 1)
	testLineDriveOption(t, gpiocdev.AsPushPull, drive, 0, 1)
	testLineDriveReconfigure(t, gpiocdev.AsOpenDrain, gpiocdev.AsPushPull, drive, 0, 1)
}

func testChipBiasOption(t *testing.T, option gpiocdev.ChipOption,
	bias gpiocdev.LineBias, expval int) {

	tf := func(t *testing.T) {
		requireKernel(t, biasKernel)

		c := getChip(t, option)
		defer c.Close()

		l, err := c.RequestLine(platform.FloatingLines()[0],
			gpiocdev.AsInput)
		assert.Nil(t, err)
		require.NotNil(t, l)
		defer l.Close()
		inf, err := c.LineInfo(platform.FloatingLines()[0])
		assert.Nil(t, err)
		assert.Equal(t, bias, inf.Config.Bias)

		if expval == -1 {
			return
		}
		v, err := l.Value()
		assert.Nil(t, err)
		assert.Equal(t, expval, v)
	}
	t.Run("Chip", tf)
}

func testLineBiasOption(t *testing.T, option gpiocdev.LineReqOption,
	bias gpiocdev.LineBias, expval int) {

	tf := func(t *testing.T) {
		requireKernel(t, biasKernel)

		c := getChip(t)
		defer c.Close()
		l, err := c.RequestLine(platform.FloatingLines()[0],
			gpiocdev.AsInput, option)
		assert.Nil(t, err)
		require.NotNil(t, l)
		defer l.Close()
		inf, err := c.LineInfo(platform.FloatingLines()[0])
		assert.Nil(t, err)
		assert.Equal(t, bias, inf.Config.Bias)
		if expval == -1 {
			return
		}
		v, err := l.Value()
		assert.Nil(t, err)
		assert.Equal(t, expval, v)
	}
	t.Run("Line", tf)
}

func testLineBiasReconfigure(t *testing.T, createOption gpiocdev.LineReqOption,
	reconfigOption gpiocdev.LineConfigOption, bias gpiocdev.LineBias, expval int) {

	tf := func(t *testing.T) {
		requireKernel(t, setConfigKernel)

		c := getChip(t)
		defer c.Close()
		l, err := c.RequestLine(platform.FloatingLines()[0],
			createOption, gpiocdev.AsInput)
		assert.Nil(t, err)
		require.NotNil(t, l)
		defer l.Close()
		l.Reconfigure(reconfigOption)
		inf, err := c.LineInfo(platform.FloatingLines()[0])
		assert.Nil(t, err)
		assert.Equal(t, bias, inf.Config.Bias)
		if expval == -1 {
			return
		}
		v, err := l.Value()
		assert.Nil(t, err)
		assert.Equal(t, expval, v)
	}
	t.Run("Reconfigure", tf)
}

func TestWithBiasDisabled(t *testing.T) {
	bias := gpiocdev.LineBiasDisabled
	// can't test value - is indeterminate without external bias.
	testChipBiasOption(t, gpiocdev.WithBiasDisabled, bias, -1)
	testLineBiasOption(t, gpiocdev.WithBiasDisabled, bias, -1)
	testLineBiasReconfigure(t, gpiocdev.WithPullDown, gpiocdev.WithBiasDisabled, bias, -1)
}

func TestWithBiasAsIs(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	c := getChip(t, gpiocdev.WithConsumer("TestWithBiasAsIs"))
	defer c.Close()
	requireABI(t, c, 2)

	ll := platform.FloatingLines()
	require.GreaterOrEqual(t, len(ll), 5)

	l, err := c.RequestLines(ll,
		gpiocdev.AsInput,
		gpiocdev.WithPullDown,
		gpiocdev.WithLines(
			[]int{ll[2], ll[4]},
			gpiocdev.WithBiasAsIs,
		),
	)
	assert.Nil(t, err)
	require.NotNil(t, l)

	xinf := gpiocdev.LineInfo{
		Used:     true,
		Consumer: "TestWithBiasAsIs",
		Offset:   ll[0],
		Config: gpiocdev.LineConfig{
			Bias:      gpiocdev.LineBiasPullDown,
			Direction: gpiocdev.LineDirectionInput,
		},
	}
	inf, err := c.LineInfo(ll[0])
	assert.Nil(t, err)
	xinf.Name = inf.Name // don't care about line name
	assert.Equal(t, xinf, inf)

	inf, err = c.LineInfo(ll[2])
	assert.Nil(t, err)
	xinf.Offset = ll[2]
	xinf.Config.Bias = gpiocdev.LineBiasUnknown
	xinf.Name = inf.Name // don't care about line name
	assert.Equal(t, xinf, inf)

	l.Close()
}

func TestWithPullDown(t *testing.T) {
	bias := gpiocdev.LineBiasPullDown
	testChipBiasOption(t, gpiocdev.WithPullDown, bias, 0)
	testLineBiasOption(t, gpiocdev.WithPullDown, bias, 0)
	testLineBiasReconfigure(t, gpiocdev.WithPullUp, gpiocdev.WithPullDown, bias, 0)
}
func TestWithPullUp(t *testing.T) {
	bias := gpiocdev.LineBiasPullUp
	testChipBiasOption(t, gpiocdev.WithPullUp, bias, 1)
	testLineBiasOption(t, gpiocdev.WithPullUp, bias, 1)
	testLineBiasReconfigure(t, gpiocdev.WithPullDown, gpiocdev.WithPullUp, bias, 1)
}

var evtSeqno uint32

type AbiVersioner interface {
	UapiAbiVersion() int
}

func nextEvent(r AbiVersioner, active int) gpiocdev.LineEvent {
	if r.UapiAbiVersion() != 1 {
		evtSeqno++
	}
	typ := gpiocdev.LineEventFallingEdge
	if active != 0 {
		typ = gpiocdev.LineEventRisingEdge
	}
	return gpiocdev.LineEvent{
		Type:      typ,
		Seqno:     evtSeqno,
		LineSeqno: evtSeqno,
	}
}

func TestWithEventHandler(t *testing.T) {
	platform.TriggerIntr(0)

	// via chip options
	ich := make(chan gpiocdev.LineEvent, 3)
	eh := func(evt gpiocdev.LineEvent) {
		ich <- evt
	}
	chipOpts := []gpiocdev.ChipOption{gpiocdev.WithEventHandler(eh)}
	if kernelAbiVersion != 0 {
		chipOpts = append(chipOpts, gpiocdev.ABIVersionOption(kernelAbiVersion))
	}
	c := getChip(t, chipOpts...)
	defer c.Close()

	r, err := c.RequestLine(platform.IntrLine(), gpiocdev.WithBothEdges)
	require.Nil(t, err)
	require.NotNil(t, r)
	defer r.Close()
	evtSeqno = 0
	waitNoEvent(t, ich)
	platform.TriggerIntr(1)
	waitEvent(t, ich, nextEvent(r, 1))
	platform.TriggerIntr(0)
	waitEvent(t, ich, nextEvent(r, 0))
	platform.TriggerIntr(1)
	waitEvent(t, ich, nextEvent(r, 1))
	platform.TriggerIntr(0)
	waitEvent(t, ich, nextEvent(r, 0))
	waitNoEvent(t, ich)

	r.Close()

	// via line options
	ich2 := make(chan gpiocdev.LineEvent, 3)
	eh2 := func(evt gpiocdev.LineEvent) {
		ich2 <- evt
	}
	r, err = c.RequestLine(platform.IntrLine(),
		gpiocdev.WithBothEdges,
		gpiocdev.WithEventHandler(eh2))
	require.Nil(t, err)
	require.NotNil(t, r)
	defer r.Close()
	evtSeqno = 0
	waitNoEvent(t, ich2)
	platform.TriggerIntr(1)
	waitEvent(t, ich2, nextEvent(r, 1))
	platform.TriggerIntr(0)
	waitEvent(t, ich2, nextEvent(r, 0))
	platform.TriggerIntr(1)
	waitEvent(t, ich2, nextEvent(r, 1))
	platform.TriggerIntr(0)
	waitEvent(t, ich2, nextEvent(r, 0))
	waitNoEvent(t, ich2)

	r.Close()

	// stub out inherted event handler
	r, err = c.RequestLine(platform.IntrLine(),
		gpiocdev.WithEventHandler(nil))
	require.Nil(t, err)
	require.NotNil(t, r)
	defer r.Close()
	waitNoEvent(t, ich)
	platform.TriggerIntr(1)
	waitNoEvent(t, ich)
	platform.TriggerIntr(0)
	waitNoEvent(t, ich)
}

func TestWithFallingEdge(t *testing.T) {
	platform.TriggerIntr(1)
	c := getChip(t)
	defer c.Close()

	ich := make(chan gpiocdev.LineEvent, 3)
	r, err := c.RequestLine(platform.IntrLine(),
		gpiocdev.WithFallingEdge,
		gpiocdev.WithEventHandler(func(evt gpiocdev.LineEvent) {
			ich <- evt
		}))
	require.Nil(t, err)
	require.NotNil(t, r)
	defer r.Close()
	evtSeqno = 0
	waitNoEvent(t, ich)
	platform.TriggerIntr(0)
	waitEvent(t, ich, nextEvent(r, 0))
	platform.TriggerIntr(1)
	waitNoEvent(t, ich)
	platform.TriggerIntr(0)
	waitEvent(t, ich, nextEvent(r, 0))
	platform.TriggerIntr(1)
	waitNoEvent(t, ich)
}

func TestWithRisingEdge(t *testing.T) {
	platform.TriggerIntr(0)
	c := getChip(t)
	defer c.Close()

	ich := make(chan gpiocdev.LineEvent, 3)
	r, err := c.RequestLine(platform.IntrLine(),
		gpiocdev.WithRisingEdge,
		gpiocdev.WithEventHandler(func(evt gpiocdev.LineEvent) {
			ich <- evt
		}))
	require.Nil(t, err)
	require.NotNil(t, r)
	defer r.Close()
	evtSeqno = 0
	waitNoEvent(t, ich)
	platform.TriggerIntr(1)
	waitEvent(t, ich, nextEvent(r, 1))
	platform.TriggerIntr(0)
	waitNoEvent(t, ich)
	platform.TriggerIntr(1)
	waitEvent(t, ich, nextEvent(r, 1))
	platform.TriggerIntr(0)
	waitNoEvent(t, ich)
}

func TestWithBothEdges(t *testing.T) {
	platform.TriggerIntr(0)
	c := getChip(t)
	defer c.Close()

	ich := make(chan gpiocdev.LineEvent, 3)
	lines := append(platform.FloatingLines(), platform.IntrLine())
	r, err := c.RequestLines(lines,
		gpiocdev.WithBothEdges,
		gpiocdev.WithEventHandler(func(evt gpiocdev.LineEvent) {
			ich <- evt
		}))
	require.Nil(t, err)
	require.NotNil(t, r)
	defer r.Close()
	evtSeqno = 0
	waitNoEvent(t, ich)
	platform.TriggerIntr(1)
	waitEvent(t, ich, nextEvent(r, 1))
	platform.TriggerIntr(0)
	waitEvent(t, ich, nextEvent(r, 0))
	platform.TriggerIntr(1)
	waitEvent(t, ich, nextEvent(r, 1))
	platform.TriggerIntr(0)
	waitEvent(t, ich, nextEvent(r, 0))
	waitNoEvent(t, ich)
}

func TestWithoutEdges(t *testing.T) {
	platform.TriggerIntr(0)
	c := getChip(t)
	defer c.Close()

	ich := make(chan gpiocdev.LineEvent, 3)
	lines := append(platform.FloatingLines(), platform.IntrLine())
	r, err := c.RequestLines(lines,
		gpiocdev.WithBothEdges,
		gpiocdev.WithEventHandler(func(evt gpiocdev.LineEvent) {
			ich <- evt
		}))
	require.Nil(t, err)
	require.NotNil(t, r)
	defer r.Close()
	evtSeqno = 0
	waitNoEvent(t, ich)
	platform.TriggerIntr(1)
	waitEvent(t, ich, nextEvent(r, 1))
	platform.TriggerIntr(0)
	waitEvent(t, ich, nextEvent(r, 0))

	err = r.Reconfigure(gpiocdev.WithoutEdges)
	if c.UapiAbiVersion() == 1 {
		// uapi v2 required for edge reconfiguration
		assert.Equal(t, unix.EINVAL, err)
		return
	}
	require.Nil(t, err)
	waitNoEvent(t, ich)
	platform.TriggerIntr(1)
	waitNoEvent(t, ich)
	platform.TriggerIntr(0)
	waitNoEvent(t, ich)
}

func TestWithRealtimeEventClock(t *testing.T) {
	platform.TriggerIntr(0)
	c := getChip(t)
	defer c.Close()

	var evtTimestamp time.Duration
	ich := make(chan gpiocdev.LineEvent, 3)
	lines := append(platform.FloatingLines(), platform.IntrLine())
	r, err := c.RequestLines(lines,
		gpiocdev.WithBothEdges,
		gpiocdev.WithRealtimeEventClock,
		gpiocdev.WithEventHandler(func(evt gpiocdev.LineEvent) {
			evtTimestamp = evt.Timestamp
			ich <- evt
		}))
	if c.UapiAbiVersion() == 1 {
		// uapi v2 required for event clock option
		assert.Equal(t, gpiocdev.ErrUapiIncompatibility{Feature: "event clock", AbiVersion: 1}, err)
		assert.Nil(t, r)
		return
	}
	if mockup.CheckKernelVersion(eventClockRealtimeKernel) != nil {
		// old kernels should reject the realtime request
		assert.Equal(t, unix.EINVAL, err)
		assert.Nil(t, r)
		if r != nil {
			r.Close()
		}
		return
	}
	require.Nil(t, err)
	require.NotNil(t, r)
	defer r.Close()
	evtSeqno = 0
	waitNoEvent(t, ich)
	start := time.Now()
	platform.TriggerIntr(1)
	waitEvent(t, ich, nextEvent(r, 1))
	end := time.Now()
	// with time converted to nanoseconds duration
	assert.LessOrEqual(t, start.UnixNano(), evtTimestamp.Nanoseconds())
	assert.GreaterOrEqual(t, end.UnixNano(), evtTimestamp.Nanoseconds())
	// with timestamp converted to time
	evtTime := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC).Add(evtTimestamp)
	assert.False(t, evtTime.Before(start))
	assert.False(t, evtTime.After(end))

	start = time.Now()
	platform.TriggerIntr(0)
	waitEvent(t, ich, nextEvent(r, 0))
	end = time.Now()
	assert.LessOrEqual(t, start.UnixNano(), evtTimestamp.Nanoseconds())
	assert.GreaterOrEqual(t, end.UnixNano(), evtTimestamp.Nanoseconds())
	evtTime = time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC).Add(evtTimestamp)
	assert.False(t, evtTime.Before(start))
	assert.False(t, evtTime.After(end))
}

func waitEvent(t *testing.T, ch <-chan gpiocdev.LineEvent, xevt gpiocdev.LineEvent) {
	t.Helper()
	select {
	case evt := <-ch:
		assert.Equal(t, xevt.Type, evt.Type)
		assert.Equal(t, xevt.Seqno, evt.Seqno)
		assert.Equal(t, xevt.LineSeqno, evt.LineSeqno)
	case <-time.After(time.Second):
		assert.Fail(t, "timeout waiting for event")
	}
}

func waitNoEvent(t *testing.T, ch <-chan gpiocdev.LineEvent) {
	t.Helper()
	select {
	case evt := <-ch:
		assert.Fail(t, "received unexpected event", evt)
	case <-time.After(20 * time.Millisecond):
	}
}

func clearEvents(ch <-chan gpiocdev.LineEvent) uint32 {
	var seqno uint32
	select {
	case evt := <-ch:
		seqno = evt.Seqno
	case <-time.After(20 * time.Millisecond):
	}
	return seqno
}

func TestWithDebounce(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	c := getChip(t)
	defer c.Close()

	l, err := c.RequestLine(platform.IntrLine(),
		gpiocdev.WithDebounce(10*time.Microsecond))

	if c.UapiAbiVersion() == 1 {
		xerr := gpiocdev.ErrUapiIncompatibility{"debounce", 1}
		assert.Equal(t, xerr, err)
		assert.Nil(t, l)
		return
	}
	require.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()

	inf, err := c.LineInfo(platform.IntrLine())
	assert.Nil(t, err)
	assert.Equal(t, gpiocdev.LineDirectionInput, inf.Config.Direction)
	assert.True(t, inf.Config.Debounced)
	assert.Equal(t, 10*time.Microsecond, inf.Config.DebouncePeriod)
}

func TestWithLines(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	c := getChip(t, gpiocdev.WithConsumer("TestWithLines"))
	defer c.Close()
	requireABI(t, c, 2)

	ll := platform.FloatingLines()
	require.GreaterOrEqual(t, len(ll), 5)

	patterns := []struct {
		name       string
		reqOptions []gpiocdev.LineReqOption
		info       map[int]gpiocdev.LineInfo
	}{
		{"in+out",
			[]gpiocdev.LineReqOption{
				gpiocdev.AsInput,
				gpiocdev.WithPullDown,
				gpiocdev.WithLines(
					[]int{ll[2], ll[4]},
					gpiocdev.AsOutput(1, 1),
					gpiocdev.AsActiveLow,
					gpiocdev.WithPullUp,
					gpiocdev.AsOpenDrain,
				),
			},
			map[int]gpiocdev.LineInfo{
				ll[0]: {
					Config: gpiocdev.LineConfig{
						Bias:      gpiocdev.LineBiasPullDown,
						Direction: gpiocdev.LineDirectionInput,
					},
				},
				ll[2]: {
					Config: gpiocdev.LineConfig{
						ActiveLow: true,
						Bias:      gpiocdev.LineBiasPullUp,
						Direction: gpiocdev.LineDirectionOutput,
						Drive:     gpiocdev.LineDriveOpenDrain,
					},
				},
			},
		},
		{"in+debounced",
			[]gpiocdev.LineReqOption{
				gpiocdev.AsInput,
				gpiocdev.WithLines(
					[]int{ll[2], ll[4]},
					gpiocdev.WithDebounce(1234*time.Microsecond),
				),
				gpiocdev.AsActiveLow,
			},
			map[int]gpiocdev.LineInfo{
				ll[1]: {
					Config: gpiocdev.LineConfig{
						ActiveLow: true,
						Direction: gpiocdev.LineDirectionInput,
					},
				},
				ll[4]: {
					Config: gpiocdev.LineConfig{
						Debounced:      true,
						DebouncePeriod: 1234 * time.Microsecond,
						Direction:      gpiocdev.LineDirectionInput,
					},
				},
			},
		},
		{"out+debounced",
			[]gpiocdev.LineReqOption{
				gpiocdev.AsOutput(1, 0, 1, 1),
				gpiocdev.WithLines(
					[]int{ll[2], ll[4]},
					gpiocdev.WithDebounce(1432*time.Microsecond),
				),
			},
			map[int]gpiocdev.LineInfo{
				ll[3]: {
					Config: gpiocdev.LineConfig{
						Direction: gpiocdev.LineDirectionOutput,
					},
				},
				ll[4]: {
					Config: gpiocdev.LineConfig{
						Debounced:      true,
						DebouncePeriod: 1432 * time.Microsecond,
						Direction:      gpiocdev.LineDirectionInput,
					},
				},
			},
		},
		{"debounced+debounced",
			[]gpiocdev.LineReqOption{
				gpiocdev.WithDebounce(1234 * time.Microsecond),
				gpiocdev.WithLines(
					[]int{ll[2], ll[1]},
					gpiocdev.WithDebounce(1432*time.Microsecond),
				),
			},
			map[int]gpiocdev.LineInfo{
				ll[0]: {
					Config: gpiocdev.LineConfig{
						Debounced:      true,
						DebouncePeriod: 1234 * time.Microsecond,
						Direction:      gpiocdev.LineDirectionInput,
					},
				},
				ll[1]: {
					Config: gpiocdev.LineConfig{
						Debounced:      true,
						DebouncePeriod: 1432 * time.Microsecond,
						Direction:      gpiocdev.LineDirectionInput,
					},
				},
			},
		},
		{"in+out+debounced",
			[]gpiocdev.LineReqOption{
				gpiocdev.AsInput,
				gpiocdev.WithLines(
					[]int{ll[2], ll[4]},
					gpiocdev.AsOutput(1, 1),
					gpiocdev.AsActiveLow,
					gpiocdev.WithPullUp,
					gpiocdev.AsOpenDrain,
				),
				gpiocdev.WithLines(
					[]int{ll[3], ll[4]},
					gpiocdev.WithDebounce(1432*time.Microsecond),
				),
				gpiocdev.WithPullDown,
			},
			map[int]gpiocdev.LineInfo{
				ll[0]: {
					Config: gpiocdev.LineConfig{
						Bias:      gpiocdev.LineBiasPullDown,
						Direction: gpiocdev.LineDirectionInput,
					},
				},
				ll[2]: {
					Config: gpiocdev.LineConfig{
						ActiveLow: true,
						Bias:      gpiocdev.LineBiasPullUp,
						Direction: gpiocdev.LineDirectionOutput,
						Drive:     gpiocdev.LineDriveOpenDrain,
					},
				},
				ll[3]: {
					Config: gpiocdev.LineConfig{
						Debounced:      true,
						DebouncePeriod: 1432 * time.Microsecond,
						Direction:      gpiocdev.LineDirectionInput,
					},
				},
				ll[4]: {
					Config: gpiocdev.LineConfig{
						ActiveLow:      true,
						Bias:           gpiocdev.LineBiasPullUp,
						Debounced:      true,
						DebouncePeriod: 1432 * time.Microsecond,
						Direction:      gpiocdev.LineDirectionInput,
					},
				},
			},
		},
	}

	for _, p := range patterns {
		tf := func(t *testing.T) {
			l, err := c.RequestLines(ll, p.reqOptions...)
			assert.Nil(t, err)
			require.NotNil(t, l)
			for offset, xinf := range p.info {
				inf, err := c.LineInfo(offset)
				xinf.Consumer = "TestWithLines"
				xinf.Used = true
				xinf.Offset = offset
				inf.Name = "" // don't care about line name
				assert.Nil(t, err)
				assert.Equal(t, xinf, inf, offset)
			}
			l.Close()
		}
		t.Run(p.name, tf)
	}

	for _, p := range patterns {
		tf := func(t *testing.T) {
			l, err := c.RequestLines(ll)
			assert.Nil(t, err)
			require.NotNil(t, l)
			reconfigOpts := []gpiocdev.LineConfigOption(nil)
			for _, opt := range p.reqOptions {
				// look away - hideous casting in progress
				lco, ok := interface{}(opt).(gpiocdev.LineConfigOption)
				if ok {
					reconfigOpts = append(reconfigOpts, lco)
				}
			}
			err = l.Reconfigure(reconfigOpts...)
			assert.Nil(t, err)
			for offset, xinf := range p.info {
				inf, err := c.LineInfo(offset)
				xinf.Consumer = "TestWithLines"
				xinf.Used = true
				xinf.Offset = offset
				inf.Name = "" // don't care about line name
				assert.Nil(t, err)
				assert.Equal(t, xinf, inf, offset)
			}
			l.Close()
		}
		t.Run("reconfig-"+p.name, tf)
	}
}

func TestDefaulted(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	c := getChip(t, gpiocdev.WithConsumer("TestDefaulted"))
	defer c.Close()

	ll := platform.FloatingLines()
	require.GreaterOrEqual(t, len(ll), 5)

	patterns := []struct {
		name       string
		reqOptions []gpiocdev.LineReqOption
		info       map[int]gpiocdev.LineInfo
		abi        int
	}{
		{"top level",
			[]gpiocdev.LineReqOption{
				gpiocdev.AsActiveLow,
				gpiocdev.WithPullDown,
				gpiocdev.Defaulted,
				gpiocdev.AsInput,
			},
			map[int]gpiocdev.LineInfo{
				ll[0]: {
					Config: gpiocdev.LineConfig{
						Direction: gpiocdev.LineDirectionInput,
					},
				},
				ll[2]: {
					Config: gpiocdev.LineConfig{
						Direction: gpiocdev.LineDirectionInput,
					},
				},
			},
			1,
		},
		{"WithLines",
			[]gpiocdev.LineReqOption{
				gpiocdev.AsInput,
				gpiocdev.WithLines(
					[]int{ll[2], ll[4]},
					gpiocdev.WithDebounce(1234*time.Microsecond),
				),
				gpiocdev.WithLines(
					[]int{ll[2]},
					gpiocdev.Defaulted,
				),
				gpiocdev.AsActiveLow,
			},
			map[int]gpiocdev.LineInfo{
				ll[1]: {
					Config: gpiocdev.LineConfig{
						ActiveLow: true,
						Direction: gpiocdev.LineDirectionInput,
					},
				},
				ll[2]: {
					Config: gpiocdev.LineConfig{
						ActiveLow: true,
						Direction: gpiocdev.LineDirectionInput,
					},
				},
				ll[4]: {
					Config: gpiocdev.LineConfig{
						Debounced:      true,
						DebouncePeriod: 1234 * time.Microsecond,
						Direction:      gpiocdev.LineDirectionInput,
					},
				},
			},
			2,
		},
		{"WithLines nil",
			[]gpiocdev.LineReqOption{
				gpiocdev.AsInput,
				gpiocdev.WithLines(
					[]int{ll[2], ll[4]},
					gpiocdev.WithDebounce(1234*time.Microsecond),
				),
				gpiocdev.WithLines(
					[]int(nil),
					gpiocdev.Defaulted,
				),
				gpiocdev.AsActiveLow,
			},
			map[int]gpiocdev.LineInfo{
				ll[1]: {
					Config: gpiocdev.LineConfig{
						ActiveLow: true,
						Direction: gpiocdev.LineDirectionInput,
					},
				},
				ll[2]: {
					Config: gpiocdev.LineConfig{
						ActiveLow: true,
						Direction: gpiocdev.LineDirectionInput,
					},
				},
				ll[4]: {
					Config: gpiocdev.LineConfig{
						ActiveLow: true,
						Direction: gpiocdev.LineDirectionInput,
					},
				},
			},
			2,
		},
		{"WithLines empty",
			[]gpiocdev.LineReqOption{
				gpiocdev.AsInput,
				gpiocdev.WithLines(
					[]int{ll[2], ll[4]},
					gpiocdev.WithDebounce(1234*time.Microsecond),
				),
				gpiocdev.WithLines(
					[]int{},
					gpiocdev.Defaulted,
				),
				gpiocdev.AsActiveLow,
			},
			map[int]gpiocdev.LineInfo{
				ll[1]: {
					Config: gpiocdev.LineConfig{
						ActiveLow: true,
						Direction: gpiocdev.LineDirectionInput,
					},
				},
				ll[2]: {
					Config: gpiocdev.LineConfig{
						ActiveLow: true,
						Direction: gpiocdev.LineDirectionInput,
					},
				},
				ll[4]: {
					Config: gpiocdev.LineConfig{
						ActiveLow: true,
						Direction: gpiocdev.LineDirectionInput,
					},
				},
			},
			2,
		},
	}

	for _, p := range patterns {
		tf := func(t *testing.T) {
			if c.UapiAbiVersion() < p.abi {
				t.Skip(ErrorBadABIVersion{p.abi, c.UapiAbiVersion()})
			}
			l, err := c.RequestLines(ll, p.reqOptions...)
			assert.Nil(t, err)
			require.NotNil(t, l)
			for offset, xinf := range p.info {
				inf, err := c.LineInfo(offset)
				xinf.Consumer = "TestDefaulted"
				xinf.Used = true
				xinf.Offset = offset
				inf.Name = "" // don't care about line name
				assert.Nil(t, err)
				assert.Equal(t, xinf, inf, offset)
			}
			l.Close()
		}
		t.Run(p.name, tf)
	}

	for _, p := range patterns {
		tf := func(t *testing.T) {
			if c.UapiAbiVersion() < p.abi {
				t.Skip(ErrorBadABIVersion{p.abi, c.UapiAbiVersion()})
			}
			l, err := c.RequestLines(ll)
			assert.Nil(t, err)
			require.NotNil(t, l)
			reconfigOpts := []gpiocdev.LineConfigOption(nil)
			for _, opt := range p.reqOptions {
				// look away - hideous casting in progress
				lco, ok := interface{}(opt).(gpiocdev.LineConfigOption)
				if ok {
					reconfigOpts = append(reconfigOpts, lco)
				}
			}
			err = l.Reconfigure(reconfigOpts...)
			assert.Nil(t, err)
			for offset, xinf := range p.info {
				inf, err := c.LineInfo(offset)
				xinf.Consumer = "TestDefaulted"
				xinf.Used = true
				xinf.Offset = offset
				inf.Name = "" // don't care about line name
				assert.Nil(t, err)
				assert.Equal(t, xinf, inf, offset)
			}
			l.Close()
		}
		t.Run("reconfig-"+p.name, tf)
	}
}

func TestWithEventBufferSize(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	c := getChip(t)
	defer c.Close()
	requireABI(t, c, 2)

	ll := platform.FloatingLines()

	patterns := []struct {
		name     string
		size     int
		numLines int
	}{
		{"one smaller",
			5,
			1,
		},
		{"one larger",
			25,
			1,
		},
		{"one default",
			0,
			1,
		},
		{"two smaller",
			5,
			1,
		},
		{"two larger",
			35,
			1,
		},
		{"two default",
			0,
			1,
		},
	}

	for _, p := range patterns {
		if p.numLines == 1 {
			t.Run(p.name, func(t *testing.T) {
				l, err := c.RequestLine(ll[0], gpiocdev.WithEventBufferSize(p.size))
				assert.Nil(t, err)
				require.NotNil(t, l)
				l.Close()
			})
		} else {
			t.Run(p.name, func(t *testing.T) {
				l, err := c.RequestLines(ll[:p.numLines], gpiocdev.WithEventBufferSize(p.size))
				assert.Nil(t, err)
				require.NotNil(t, l)
				l.Close()
			})
		}
	}
}
