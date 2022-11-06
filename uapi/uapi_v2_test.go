// SPDX-FileCopyrightText: 2020 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

//go:build linux
// +build linux

package uapi_test

import (
	"fmt"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/warthog618/go-gpiocdev/mockup"
	"github.com/warthog618/go-gpiocdev/uapi"
	"golang.org/x/sys/unix"
)

var (
	uapiV2Kernel             = mockup.Semver{5, 10} // uapi v2 added
	eventClockRealtimeKernel = mockup.Semver{5, 11} // add LineFlagV2EventClockRealtime
	debouncePeriod           = 5 * clkTick
)

type AttributeEncoder interface {
	Encode() uapi.LineAttribute
}

func TestGetLineInfoV2(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	reloadMockup() // test assumes clean mockups
	requireMockup(t)
	for n := 0; n < mock.Chips(); n++ {
		c, err := mock.Chip(n)
		require.Nil(t, err)
		for l := 0; l <= c.Lines; l++ {
			f := func(t *testing.T) {
				f, err := os.Open(c.DevPath)
				require.Nil(t, err)
				defer f.Close()
				xli := uapi.LineInfoV2{
					Offset: uint32(l),
					Flags:  uapi.LineFlagV2Input,
				}
				copy(xli.Name[:], fmt.Sprintf("%s-%d", c.Label, l))
				copy(xli.Consumer[:], "")
				li, err := uapi.GetLineInfoV2(f.Fd(), l)
				if l < c.Lines {
					assert.Nil(t, err)
					assert.Equal(t, xli, li)
				} else {
					assert.Equal(t, unix.EINVAL, err)
				}
			}
			t.Run(fmt.Sprintf("%s-%d", c.Name, l), f)
		}
	}
	// badfd
	li, err := uapi.GetLineInfoV2(0, 1)
	lix := uapi.LineInfoV2{}
	assert.NotNil(t, err)
	assert.Equal(t, lix, li)
}

func TestGetLine(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	reloadMockup()
	requireMockup(t)
	patterns := []struct {
		name string // unique name for pattern (hf/ef/offsets/xval combo)
		cnum int
		lr   uapi.LineRequest
		err  error
	}{
		{
			"as-is",
			0,
			uapi.LineRequest{
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
		},
		{
			"atv-lo",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2ActiveLow,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
		},
		{
			"input",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
		},
		{
			"input pull-up",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input | uapi.LineFlagV2BiasPullUp,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
		},
		{
			"input pull-down",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input | uapi.LineFlagV2BiasPullDown,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{3},
			},
			nil,
		},
		{
			"input bias disable",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input | uapi.LineFlagV2BiasDisabled,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{3},
			},
			nil,
		},
		{
			"input edge rising",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeRising,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{3},
			},
			nil,
		},
		{
			"input edge falling",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeFalling,
				},
				Lines:   2,
				Offsets: [uapi.LinesMax]uint32{1, 3},
			},
			nil,
		},
		{
			"input edge both",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeBoth,
				},
				Lines:           1,
				Offsets:         [uapi.LinesMax]uint32{3},
				EventBufferSize: 42,
			},
			nil,
		},
		{
			"output",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
		},
		{
			"output drain",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output | uapi.LineFlagV2OpenDrain,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
		},
		{
			"output source",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output | uapi.LineFlagV2OpenSource,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
		},
		{
			"output pull-up",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output | uapi.LineFlagV2BiasPullUp,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			nil,
		},
		{
			"output pull-down",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output | uapi.LineFlagV2BiasPullDown,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
		},
		{
			"output bias disabled",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output | uapi.LineFlagV2BiasDisabled,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			nil,
		},
		// expected errors
		{
			"overlength",
			0,
			uapi.LineRequest{
				Lines:   5,
				Offsets: [uapi.LinesMax]uint32{0, 1, 2, 3, 4},
			},
			unix.EINVAL,
		},
	}
	for _, p := range patterns {
		c, err := mock.Chip(p.cnum)
		require.Nil(t, err)
		tf := func(t *testing.T) {
			f, err := os.Open(c.DevPath)
			require.Nil(t, err)
			defer f.Close()
			copy(p.lr.Consumer[:], p.name)
			err = uapi.GetLine(f.Fd(), &p.lr)
			assert.Equal(t, p.err, err)
			if p.lr.Offsets[0] > uint32(c.Lines) {
				return
			}
			// check line info
			li, err := uapi.GetLineInfoV2(f.Fd(), int(p.lr.Offsets[0]))
			assert.Nil(t, err)
			if p.err != nil {
				assert.True(t, li.Flags.IsAvailable())
				unix.Close(int(p.lr.Fd))
				return
			}
			xli := uapi.LineInfoV2{
				Offset: p.lr.Offsets[0],
				Flags:  uapi.LineFlagV2Used | p.lr.Config.Flags,
			}
			copy(xli.Name[:], li.Name[:]) // don't care about name
			copy(xli.Consumer[:31], p.name)
			if xli.Flags&uapi.LineFlagV2DirectionMask == 0 {
				xli.Flags |= uapi.LineFlagV2Input
			}
			assert.Equal(t, xli, li)
			unix.Close(int(p.lr.Fd))
		}
		t.Run(p.name, tf)
	}
}

func TestGetLineValidation(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	requireMockup(t)
	patterns := []struct {
		name string
		lr   uapi.LineRequest
	}{
		{
			"oorange offset",
			uapi.LineRequest{
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{6},
			},
		},
		{
			"input drain",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input | uapi.LineFlagV2OpenDrain,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
		},
		{
			"input source",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input | uapi.LineFlagV2OpenSource,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
		},
		{
			"as-is drain",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2OpenDrain,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
		},
		{
			"as-is source",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2OpenSource,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
		},
		{
			"as-is pull-up",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2BiasPullUp,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
		},
		{
			"as-is pull-down",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2BiasPullDown,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
		},
		{
			"as-is bias disabled",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2BiasDisabled,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
		},
		{
			"output edge rising",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output | uapi.LineFlagV2EdgeRising,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
		},
		{
			"output edge falling",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output | uapi.LineFlagV2EdgeFalling,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
		},
		{
			"output edge both",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output | uapi.LineFlagV2EdgeBoth,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
		},
		{
			"as-is edge rising",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2EdgeRising,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
		},
		{
			"as-is edge falling",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2EdgeFalling,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
		},
		{
			"as-is edge both",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2EdgeBoth,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
		},
		{
			"non-zero padding",
			uapi.LineRequest{
				Config:  uapi.LineConfig{Padding: [5]uint32{1}},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
		},
	}
	c, err := mock.Chip(0)
	require.Nil(t, err)
	f, err := os.Open(c.DevPath)
	require.Nil(t, err)
	defer f.Close()
	for _, p := range patterns {
		tf := func(t *testing.T) {
			copy(p.lr.Consumer[:31], "test-get-line-validation")
			err = uapi.GetLine(f.Fd(), &p.lr)
			assert.Equal(t, unix.EINVAL, err)
		}
		t.Run(p.name, tf)
	}
}

func TestGetLineValuesV2(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	requireMockup(t)
	patterns := []struct {
		name   string
		cnum   int
		lr     uapi.LineRequest
		active []int
		mask   []int
		err    error
	}{
		{
			"as-is atv-lo lo",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2ActiveLow,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{},
			[]int{0},
			nil,
		},
		{
			"as-is atv-lo hi",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2ActiveLow,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{0},
			[]int{0},
			nil,
		},
		{
			"as-is lo",
			0,
			uapi.LineRequest{
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{},
			[]int{0},
			nil,
		},
		{
			"as-is hi",
			0,
			uapi.LineRequest{
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			[]int{0},
			[]int{0},
			nil,
		},
		{
			"input lo",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{},
			[]int{0},
			nil,
		},
		{
			"input hi",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			[]int{0},
			[]int{0},
			nil,
		},
		{
			"output lo",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{},
			[]int{0},
			nil,
		},
		{
			"output hi",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			[]int{0},
			[]int{0},
			nil,
		},
		{
			"both lo",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeBoth,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{},
			[]int{0},
			nil,
		},
		{
			"both hi",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeBoth,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			[]int{0},
			[]int{0},
			nil,
		},
		{
			"falling lo",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeFalling,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{},
			[]int{0},
			nil,
		},
		{
			"falling hi",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeFalling,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			[]int{0},
			[]int{0},
			nil,
		},
		{
			"rising lo",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeRising,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{},
			[]int{0},
			nil,
		},
		{
			"rising hi",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeRising,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			[]int{0},
			[]int{0},
			nil,
		},
		{
			"input 2a",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input,
				},
				Lines:   2,
				Offsets: [uapi.LinesMax]uint32{0, 1},
			},
			[]int{0},
			[]int{0, 1},
			nil,
		},
		{
			"input 2b",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input,
				},
				Lines:   2,
				Offsets: [uapi.LinesMax]uint32{2, 1},
			},
			[]int{1},
			[]int{0, 1},
			nil,
		},
		{
			"input 3a",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{0, 1, 2},
			},
			[]int{1, 2},
			[]int{0, 1, 2},
			nil,
		},
		{
			"input 3b",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{0, 2, 1},
			},
			[]int{1},
			[]int{0, 1, 2},
			nil,
		},
		{
			"input 4a",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input,
				},
				Lines:   4,
				Offsets: [uapi.LinesMax]uint32{0, 1, 2, 3},
			},
			[]int{1, 2, 3},
			[]int{0, 1, 2, 3},
			nil,
		},
		{
			"input 4b",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input,
				},
				Lines:   4,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0},
			},
			[]int{0, 1, 3},
			[]int{0, 1, 2, 3},
			nil,
		},
		{
			"input 8a",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{0, 1, 2, 3, 4, 5, 6, 7},
			},
			[]int{1, 2, 3, 4, 5},
			[]int{0, 1, 2, 3, 4, 5, 6, 7},
			nil,
		},
		{
			"input 8b",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]int{0, 1, 3, 4, 5, 7},
			[]int{0, 1, 2, 3, 4, 5, 6, 7},
			nil,
		},
		{
			"atv-lo 8b",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2ActiveLow,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]int{0, 1, 3, 4, 5},
			[]int{0, 1, 2, 3, 4, 5, 6, 7},
			nil,
		},
		{
			"sparse atv-lo",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2ActiveLow,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]int{0, 1, 3, 4, 5},
			[]int{0, 2, 3, 5, 7},
			nil,
		},
		{
			"sparse atv-hi",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]int{0, 1, 3, 4, 5},
			[]int{0, 2, 3, 5, 7},
			nil,
		},
		{
			"sparse one lo",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]int{0, 1, 3, 4, 5},
			[]int{2},
			nil,
		},
		{
			"sparse one hi",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]int{0, 1, 3, 4, 5},
			[]int{3},
			nil,
		},
		{
			"overwide sparse atv-hi",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]int{0, 1, 3, 4, 5},
			[]int{0, 2, 3, 5, 7, 8, 9, 10, 11},
			nil,
		},
		{
			"edge detection lo",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeFalling,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{},
			[]int{0},
			nil,
		},
		{
			"edge detection hi",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeBoth,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			[]int{0},
			[]int{0},
			nil,
		},
		{
			"zero mask",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeBoth,
				},
				Lines:   4,
				Offsets: [uapi.LinesMax]uint32{0, 1, 2, 3},
			},
			[]int{0, 2, 3},
			[]int{},
			unix.EINVAL,
		},
	}
	for _, p := range patterns {
		c, err := mock.Chip(p.cnum)
		require.Nil(t, err)
		// set vals in mock
		vals := uapi.NewLineBits(p.active...)
		for i := 0; i < int(p.lr.Lines); i++ {
			v := vals.Get(i)
			o := int(p.lr.Offsets[i])
			if p.lr.Config.Flags.IsActiveLow() {
				v ^= 0x01
			}
			err := c.SetValue(o, v)
			assert.Nil(t, err)
		}
		tf := func(t *testing.T) {
			f, err := os.Open(c.DevPath)
			require.Nil(t, err)
			defer f.Close()
			var fd int32
			mask := uapi.NewLineBits(p.mask...)
			xval := vals & mask
			copy(p.lr.Consumer[:31], "test-get-line-values-V2")
			err = uapi.GetLine(f.Fd(), &p.lr)
			require.Nil(t, err)
			fd = p.lr.Fd
			if p.lr.Config.Flags.IsOutput() {
				// mock is ignored for outputs
				xval = 0
			}
			lvx := uapi.LineValues{
				Mask: mask,
				Bits: xval,
			}
			lv := uapi.LineValues{
				Mask: mask,
				Bits: uapi.NewLineBits(p.active...),
			}
			err = uapi.GetLineValuesV2(uintptr(fd), &lv)
			assert.Equal(t, p.err, err)
			if p.err == nil {
				assert.Equal(t, lvx, lv)
			}
			unix.Close(int(fd))
		}
		t.Run(p.name, tf)
	}
	// badfd
	lvx := uapi.LineValues{
		Mask: uapi.NewLineBitMask(3),
	}
	lv := uapi.LineValues{
		Mask: uapi.NewLineBitMask(3),
	}
	err := uapi.GetLineValuesV2(0, &lv)
	assert.NotNil(t, err)
	assert.Equal(t, lvx, lv)
}

func TestSetLineValuesV2(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	requireMockup(t)
	patterns := []struct {
		name   string
		cnum   int
		lr     uapi.LineRequest
		active []int
		mask   []int
		err    error
	}{
		{
			"output atv-lo lo",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2ActiveLow | uapi.LineFlagV2Output,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{},
			[]int{0},
			nil,
		},
		{
			"output atv-lo hi",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2ActiveLow | uapi.LineFlagV2Output,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{0},
			[]int{0},
			nil,
		},
		{
			"as-is lo",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2ActiveLow,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{},
			[]int{0},
			nil,
		},
		{
			"as-is hi",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2ActiveLow,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{0},
			[]int{0},
			nil,
		},
		{
			"output lo",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{},
			[]int{0},
			nil,
		},
		{
			"output hi",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			[]int{0},
			[]int{0},
			nil,
		},
		{
			"output 2a",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output,
				},
				Lines:   2,
				Offsets: [uapi.LinesMax]uint32{0, 1},
			},
			[]int{0},
			[]int{0, 1},
			nil,
		},
		{
			"output 2b",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output,
				},
				Lines:   2,
				Offsets: [uapi.LinesMax]uint32{2, 1},
			},
			[]int{1},
			[]int{0, 1},
			nil,
		},
		{
			"output 3a",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{0, 1, 2},
			},
			[]int{1, 2},
			[]int{0, 1, 2},
			nil,
		},
		{
			"output 3b",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{0, 2, 1},
			},
			[]int{1},
			[]int{0, 1, 2},
			nil,
		},
		{
			"output 4a",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output,
				},
				Lines:   4,
				Offsets: [uapi.LinesMax]uint32{0, 1, 2, 3},
			},
			[]int{1, 2, 3},
			[]int{0, 1, 2, 3},
			nil,
		},
		{
			"output 4b",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output,
				},
				Lines:   4,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0},
			},
			[]int{0, 1, 3},
			[]int{0, 1, 2, 3},
			nil,
		},
		{
			"output 8a",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{0, 1, 2, 3, 4, 5, 6, 7},
			},
			[]int{1, 2, 3, 4, 5},
			[]int{0, 1, 2, 3, 4, 5, 6, 7},
			nil,
		},
		{
			"output 8b",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]int{0, 1, 3, 4, 5, 7},
			[]int{0, 1, 2, 3, 4, 5, 6, 7},
			nil,
		},
		{
			"atv-lo 8b",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2ActiveLow,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]int{0, 1, 3, 4},
			[]int{0, 1, 2, 3, 4, 5, 6, 7},
			nil,
		},
		{
			"sparse atv-hi",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]int{1, 2, 3, 4, 5},
			[]int{0, 2, 4, 5, 7},
			nil,
		},
		{
			"sparse one lo",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]int{1, 2, 3, 5},
			[]int{4},
			nil,
		},
		{
			"sparse one hi",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]int{1, 2, 3, 4, 5},
			[]int{4},
			nil,
		},
		{
			"overwide sparse atv-hi",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]int{1, 2, 3, 4, 5},
			[]int{0, 2, 4, 5, 7, 8, 9, 10, 11},
			nil,
		},
		{
			"sparse atv-lo",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output | uapi.LineFlagV2ActiveLow,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]int{1, 2, 3, 4, 5},
			[]int{0, 2, 4, 5, 7},
			nil,
		},
		// expected failures....
		{
			"input lo",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{},
			[]int{0},
			unix.EPERM,
		},
		{
			"input hi",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			[]int{0},
			[]int{0},
			unix.EPERM,
		},
		{
			"edge detection",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeRising,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{},
			[]int{0},
			unix.EPERM,
		},
		{
			"zero mask",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]int{0, 1, 3, 4, 5, 7},
			[]int{},
			unix.EINVAL,
		},
	}
	for _, p := range patterns {
		tf := func(t *testing.T) {
			c, err := mock.Chip(p.cnum)
			require.Nil(t, err)
			f, err := os.Open(c.DevPath)
			require.Nil(t, err)
			defer f.Close()
			copy(p.lr.Consumer[:31], "test-set-line-values-V2")
			err = uapi.GetLine(f.Fd(), &p.lr)
			require.Nil(t, err)
			lv := uapi.NewLineBits(p.active...)
			mask := uapi.NewLineBits(p.mask...)
			xlv := lv & mask
			lsv := uapi.LineValues{
				Mask: mask,
				Bits: lv,
			}
			err = uapi.SetLineValuesV2(uintptr(p.lr.Fd), lsv)
			assert.Equal(t, p.err, err)
			if p.err == nil {
				// check values from mock
				for i := 0; i < int(p.lr.Lines); i++ {
					o := int(p.lr.Offsets[i])
					v, err := c.Value(int(o))
					assert.Nil(t, err)
					xv := xlv.Get(i)
					if p.lr.Config.Flags.IsActiveLow() {
						xv ^= 0x01
					}
					assert.Equal(t, xv, v)
				}
			}
			unix.Close(int(p.lr.Fd))
		}
		t.Run(p.name, tf)
	}
	// badfd
	err := uapi.SetLineValuesV2(0,
		uapi.LineValues{
			Mask: 1,
			Bits: 1,
		})
	assert.NotNil(t, err)
}

func zeroed(data []byte) bool {
	for _, d := range data {
		if d != 0 {
			return false
		}
	}
	return true
}

func TestSetLineConfigV2(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	requireMockup(t)
	patterns := []struct {
		name   string
		cnum   int
		lr     uapi.LineRequest
		ra     []AttributeEncoder
		config uapi.LineConfig
		ca     []AttributeEncoder
		err    error
	}{
		{
			"in to out",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Output,
			},
			[]AttributeEncoder{uapi.OutputValues(uapi.NewLineBits(0, 2))},
			nil,
		},
		{
			"out to in",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			[]AttributeEncoder{uapi.OutputValues(uapi.NewLineBits(0, 2))},
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Input,
			},
			nil,
			nil,
		},
		{
			"as-is atv-hi to as-is atv-lo",
			0,
			uapi.LineRequest{
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagV2ActiveLow,
			},
			nil,
			nil,
		},
		{
			"as-is atv-lo to as-is atv-hi",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2ActiveLow,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{},
			nil,
			nil,
		},
		{
			"input atv-lo to input atv-hi",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input | uapi.LineFlagV2ActiveLow,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Input,
			},
			nil,
			nil,
		},
		{
			"input atv-hi to input atv-lo",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Input | uapi.LineFlagV2ActiveLow,
			},
			nil,
			nil,
		},
		{
			"output atv-lo to output atv-hi",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output | uapi.LineFlagV2ActiveLow,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			[]AttributeEncoder{uapi.OutputValues(uapi.NewLineBits(0, 0, 1))},
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Output,
			},
			[]AttributeEncoder{uapi.OutputValues(uapi.NewLineBits(0, 2))},
			nil,
		},
		{
			"output atv-hi to output atv-lo",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			[]AttributeEncoder{uapi.OutputValues(uapi.NewLineBits(0, 0, 1))},
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Output | uapi.LineFlagV2ActiveLow,
			},
			[]AttributeEncoder{uapi.OutputValues(uapi.NewLineBits(0, 2))},
			nil,
		},
		{
			"input atv-lo to as-is atv-hi",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input | uapi.LineFlagV2ActiveLow,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{},
			nil,
			nil,
		},
		{
			"input atv-hi to as-is atv-lo",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagV2ActiveLow,
			},
			nil,
			nil,
		},
		{
			"input pull-up to input pull-down",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input | uapi.LineFlagV2BiasPullUp,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Input | uapi.LineFlagV2BiasPullDown,
			},
			nil,
			nil,
		},
		{
			"input pull-down to input pull-up",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input | uapi.LineFlagV2BiasPullDown,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Input | uapi.LineFlagV2BiasPullUp,
			},
			nil,
			nil,
		},
		{
			"output atv-lo to as-is atv-hi",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output | uapi.LineFlagV2ActiveLow,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			[]AttributeEncoder{uapi.OutputValues(uapi.NewLineBits(0, 2))},
			uapi.LineConfig{},
			nil,
			nil,
		},
		{
			"output atv-hi to as-is atv-lo",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			[]AttributeEncoder{uapi.OutputValues(uapi.NewLineBits(0, 2))},
			uapi.LineConfig{
				Flags: uapi.LineFlagV2ActiveLow,
			},
			nil,
			nil,
		},
		{
			"edge to biased",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeFalling,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeFalling | uapi.LineFlagV2BiasPullUp,
			},
			nil,
			nil,
		},
		{
			"in to debounced",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Input,
			},
			[]AttributeEncoder{uapi.DebouncePeriod(20 * time.Microsecond)},
			nil,
		},
		{
			"debounced to undebounced",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			[]AttributeEncoder{uapi.DebouncePeriod(20 * time.Microsecond)},
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Input,
			},
			[]AttributeEncoder{uapi.DebouncePeriod(0)},
			nil,
		},
		{
			"debounce changed",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			[]AttributeEncoder{uapi.DebouncePeriod(20 * time.Microsecond)},
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Input,
			},
			[]AttributeEncoder{uapi.DebouncePeriod(20 * time.Microsecond)},
			nil,
		},
		{
			"out to debounced in",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			[]AttributeEncoder{uapi.OutputValues(uapi.NewLineBits(0, 2))},
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Input,
			},
			[]AttributeEncoder{uapi.DebouncePeriod(20 * time.Microsecond)},
			nil,
		},
		{
			"debounced in to out",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			[]AttributeEncoder{uapi.DebouncePeriod(20 * time.Microsecond)},
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Output,
			},
			[]AttributeEncoder{uapi.OutputValues(uapi.NewLineBits(2))},
			nil,
		},
		{
			"edge to no edge",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeFalling,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Input,
			},
			nil,
			nil,
		},
		{
			"edge to none",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeFalling,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Input,
			},
			nil,
			nil,
		},
		{
			"rising edge to falling edge",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeFalling,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeRising,
			},
			nil,
			nil,
		},
		{
			"output to edge",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Output,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeFalling,
			},
			[]AttributeEncoder{uapi.OutputValues(uapi.NewLineBits(0, 2))},
			nil,
		},
		{
			"edge to output",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeFalling,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Output,
			},
			[]AttributeEncoder{uapi.OutputValues(uapi.NewLineBits(0, 2))},
			nil,
		},
		{
			"edge to atv-lo",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeFalling,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeFalling | uapi.LineFlagV2ActiveLow,
			},
			nil,
			nil,
		},
		{
			"edge atv-lo to atv-hi",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeFalling | uapi.LineFlagV2ActiveLow,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeFalling,
			},
			nil,
			nil,
		},
		// expected errors
		{
			"input drain",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Input | uapi.LineFlagV2OpenDrain,
			},
			nil,
			unix.EINVAL,
		},
		{
			"input source",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Input | uapi.LineFlagV2OpenSource,
			},
			nil,
			unix.EINVAL,
		},
		{
			"as-is drain",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagV2OpenDrain,
			},
			nil,
			unix.EINVAL,
		},
		{
			"as-is source",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Input,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagV2OpenSource,
			},
			nil,
			unix.EINVAL,
		},
	}
	for _, p := range patterns {
		tf := func(t *testing.T) {
			c, err := mock.Chip(p.cnum)
			require.Nil(t, err)
			// setup mockup for inputs
			if p.lr.Config.Flags.IsOutput() {
				for i := uint(0); i < uint(p.lr.Lines); i++ {
					v := p.lr.Config.Attrs[0].Attr.Value64() >> i & 1
					// read is after config, so use config active state
					if p.config.Flags.IsActiveLow() {
						v ^= 0x01 // assumes using 1 for high
					}
					err := c.SetValue(int(p.lr.Offsets[i]), int(v))
					assert.Nil(t, err)
				}
			}
			f, err := os.Open(c.DevPath)
			require.Nil(t, err)
			defer f.Close()
			copy(p.lr.Consumer[:31], p.name)
			p.lr.Config.NumAttrs = uint32(len(p.ra))
			for i, a := range p.ra {
				p.lr.Config.Attrs[i].Mask = 7 // for 3 lines
				p.lr.Config.Attrs[i].Attr = a.Encode()
			}
			err = uapi.GetLine(f.Fd(), &p.lr)
			require.Nil(t, err)
			defer unix.Close(int(p.lr.Fd))
			// apply config change
			p.config.NumAttrs = uint32(len(p.ca))
			for i, a := range p.ca {
				p.config.Attrs[i].Mask = 7 // for 3 lines
				p.config.Attrs[i].Attr = a.Encode()
			}
			err = uapi.SetLineConfigV2(uintptr(p.lr.Fd), &p.config)
			require.Equal(t, p.err, err)

			if p.err == nil {
				// check line info
				li, err := uapi.GetLineInfoV2(f.Fd(), int(p.lr.Offsets[0]))
				assert.Nil(t, err)
				if p.err != nil {
					assert.False(t, li.Flags.IsUsed())
					return
				}
				xli := uapi.LineInfoV2{
					Offset: p.lr.Offsets[0],
					Flags:  uapi.LineFlagV2Used | p.config.Flags,
				}
				if p.config.Flags&uapi.LineFlagV2DirectionMask == 0 {
					xli.Flags |= p.lr.Config.Flags & uapi.LineFlagV2DirectionMask
				}
				if xli.Flags&uapi.LineFlagV2DirectionMask == 0 {
					li.Flags &^= uapi.LineFlagV2DirectionMask
				}
				for _, a := range p.ca {
					enc := a.Encode()
					if enc.ID == uapi.LineAttributeIDDebounce && !zeroed(enc.Value[:]) {
						xli.Attrs[xli.NumAttrs] = enc
						xli.NumAttrs++
					}
				}
				copy(xli.Name[:], li.Name[:]) // don't care about name
				copy(xli.Consumer[:31], p.name)
				assert.Equal(t, xli, li)
				// check values from mock
				if p.config.Flags.IsOutput() {
					for i := uint(0); i < uint(p.lr.Lines); i++ {
						v, err := c.Value(int(p.lr.Offsets[i]))
						assert.Nil(t, err)
						xv := int(p.config.Attrs[0].Attr.Value64()>>i) & 1
						if p.config.Flags.IsActiveLow() {
							xv ^= 0x01 // assumes using 1 for high
						}
						assert.Equal(t, xv, v, i)
					}
				}
			}
		}
		t.Run(p.name, tf)
	}
}

func TestSetLineConfigV2Validation(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	requireMockup(t)
	patterns := []struct {
		name string
		lc   uapi.LineConfig
	}{
		{
			"input drain",
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Input | uapi.LineFlagV2OpenDrain,
			},
		},
		{
			"input source",
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Input | uapi.LineFlagV2OpenSource,
			},
		},
		{
			"as-is drain",
			uapi.LineConfig{
				Flags: uapi.LineFlagV2OpenDrain,
			},
		},
		{
			"as-is source",
			uapi.LineConfig{
				Flags: uapi.LineFlagV2OpenSource,
			},
		},
		{
			"as-is pull-up",
			uapi.LineConfig{
				Flags: uapi.LineFlagV2BiasPullUp,
			},
		},
		{
			"as-is pull-down",
			uapi.LineConfig{
				Flags: uapi.LineFlagV2BiasPullDown,
			},
		},
		{
			"as-is bias disabled",
			uapi.LineConfig{
				Flags: uapi.LineFlagV2BiasDisabled,
			},
		},
		{
			"output edge",
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Output | uapi.LineFlagV2EdgeBoth,
			},
		},
		{
			"as-is edge",
			uapi.LineConfig{
				Flags: uapi.LineFlagV2EdgeBoth,
			},
		},
		{
			"non-zero padding",
			uapi.LineConfig{Padding: [5]uint32{1}},
		},
	}
	c, err := mock.Chip(0)
	require.Nil(t, err)
	f, err := os.Open(c.DevPath)
	require.Nil(t, err)
	defer f.Close()
	lr := uapi.LineRequest{
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagV2Input,
		},
		Lines:   1,
		Offsets: [uapi.LinesMax]uint32{2},
	}
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)
	defer unix.Close(int(lr.Fd))

	for _, p := range patterns {
		tf := func(t *testing.T) {
			err = uapi.SetLineConfigV2(uintptr(lr.Fd), &p.lc)
			assert.Equal(t, unix.EINVAL, err)
		}
		t.Run(p.name, tf)
	}
}

func TestWatchLineInfoV2(t *testing.T) {
	// also covers ReadLineInfoChangedV2

	requireKernel(t, uapiV2Kernel)
	requireMockup(t)
	c, err := mock.Chip(0)
	require.Nil(t, err)

	f, err := os.Open(c.DevPath)
	require.Nil(t, err)
	defer f.Close()

	// unwatched
	lr := uapi.LineRequest{
		Lines: 1,
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagV2Input,
		},
		Offsets: [uapi.LinesMax]uint32{3},
	}
	copy(lr.Consumer[:], "testwatch")
	err = uapi.GetLine(f.Fd(), &lr)
	assert.Nil(t, err)
	chg, err := readLineInfoChangedV2Timeout(f.Fd(), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change")
	unix.Close(int(lr.Fd))

	// out of range
	li := uapi.LineInfoV2{Offset: uint32(c.Lines + 1)}
	err = uapi.WatchLineInfoV2(f.Fd(), &li)
	require.Equal(t, syscall.Errno(0x16), err)

	// non-zero pad
	li = uapi.LineInfoV2{
		Offset:  3,
		Padding: [4]uint32{1},
	}
	err = uapi.WatchLineInfoV2(f.Fd(), &li)
	require.Equal(t, syscall.Errno(0x16), err)

	// set watch
	li = uapi.LineInfoV2{Offset: 3}
	lname := c.Label + "-3"
	err = uapi.WatchLineInfoV2(f.Fd(), &li)
	require.Nil(t, err)
	xli := uapi.LineInfoV2{
		Offset: 3,
		Flags:  uapi.LineFlagV2Input,
	}
	copy(xli.Name[:], lname)
	assert.Equal(t, xli, li)

	// repeated watch
	err = uapi.WatchLineInfoV2(f.Fd(), &li)
	assert.Equal(t, unix.EBUSY, err)

	chg, err = readLineInfoChangedV2Timeout(f.Fd(), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change")

	// request line
	lr = uapi.LineRequest{
		Lines: 1,
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagV2Input,
		},
		Offsets: [uapi.LinesMax]uint32{3},
	}
	copy(lr.Consumer[:], "testwatch")
	err = uapi.GetLine(f.Fd(), &lr)
	assert.Nil(t, err)
	chg, err = readLineInfoChangedV2Timeout(f.Fd(), eventWaitTimeout)
	assert.Nil(t, err)
	require.NotNil(t, chg)
	assert.Equal(t, uapi.LineChangedRequested, chg.Type)
	xli.Flags |= uapi.LineFlagV2Used
	copy(xli.Consumer[:], "testwatch")
	assert.Equal(t, xli, chg.Info)

	chg, err = readLineInfoChangedV2Timeout(f.Fd(), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change")

	// reconfig line
	lc := uapi.LineConfig{Flags: uapi.LineFlagV2ActiveLow}
	err = uapi.SetLineConfigV2(uintptr(lr.Fd), &lc)
	assert.Nil(t, err)
	chg, err = readLineInfoChangedV2Timeout(f.Fd(), eventWaitTimeout)
	assert.Nil(t, err)
	require.NotNil(t, chg)
	assert.Equal(t, uapi.LineChangedConfig, chg.Type)
	xli.Flags |= uapi.LineFlagV2ActiveLow
	assert.Equal(t, xli, chg.Info)

	chg, err = readLineInfoChangedV2Timeout(f.Fd(), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change")

	// release line
	unix.Close(int(lr.Fd))
	chg, err = readLineInfoChangedV2Timeout(f.Fd(), eventWaitTimeout)
	assert.Nil(t, err)
	require.NotNil(t, chg)
	assert.Equal(t, uapi.LineChangedReleased, chg.Type)
	xli = uapi.LineInfoV2{
		Offset: 3,
		Flags:  uapi.LineFlagV2Input,
	}
	copy(xli.Name[:], lname)
	assert.Equal(t, xli, chg.Info)

	chg, err = readLineInfoChangedV2Timeout(f.Fd(), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change")
}

func TestReadLineEvent(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	requireMockup(t)
	c, err := mock.Chip(0)
	require.Nil(t, err)
	f, err := os.Open(c.DevPath)
	require.Nil(t, err)
	defer f.Close()
	err = c.SetValue(1, 0)
	require.Nil(t, err)
	err = c.SetValue(2, 1)
	require.Nil(t, err)

	// active low, both edges
	lr := uapi.LineRequest{
		Lines:   2,
		Offsets: [uapi.LinesMax]uint32{1, 2},
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagV2ActiveLow | uapi.LineFlagV2Input | uapi.LineFlagV2EdgeBoth,
		},
	}
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)

	evt, err := readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	xevt := uapi.LineEvent{
		Seqno:     1,
		LineSeqno: 1,
	}

	c.SetValue(1, 1)
	evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
	require.Nil(t, err)
	require.NotNil(t, evt)
	evt.Timestamp = 0
	xevt.ID = uapi.LineEventFallingEdge
	xevt.Offset = 1
	assert.Equal(t, xevt, *evt)

	c.SetValue(2, 0)
	evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
	assert.Nil(t, err)
	require.NotNil(t, evt)
	evt.Timestamp = 0
	xevt.ID = uapi.LineEventRisingEdge
	xevt.Offset = 2
	xevt.Seqno++
	assert.Equal(t, xevt, *evt)

	c.SetValue(2, 1)
	evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
	require.Nil(t, err)
	require.NotNil(t, evt)
	evt.Timestamp = 0
	xevt.ID = uapi.LineEventFallingEdge
	xevt.Seqno++
	xevt.LineSeqno++
	assert.Equal(t, xevt, *evt)

	c.SetValue(1, 0)
	evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
	assert.Nil(t, err)
	require.NotNil(t, evt)
	evt.Timestamp = 0
	xevt.ID = uapi.LineEventRisingEdge
	xevt.Seqno++
	xevt.Offset = 1
	assert.Equal(t, xevt, *evt)

	unix.Close(int(lr.Fd))

	// falling edge
	lr.Config.Flags &^= uapi.LineFlagV2EdgeRising
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)

	c.SetValue(1, 1)
	evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
	require.Nil(t, err)
	require.NotNil(t, evt)
	evt.Timestamp = 0
	xevt.ID = uapi.LineEventFallingEdge
	xevt.Seqno = 1
	xevt.LineSeqno = 1
	xevt.Offset = 1
	assert.Equal(t, xevt, *evt)

	c.SetValue(1, 0)
	evt, err = readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	unix.Close(int(lr.Fd))

	// active hi, rising edge
	lr.Lines = 1
	lr.Config.Flags &^= uapi.LineFlagV2ActiveLow | uapi.LineFlagV2EdgeMask
	lr.Config.Flags |= uapi.LineFlagV2EdgeRising
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)

	c.SetValue(1, 1)
	evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
	require.Nil(t, err)
	require.NotNil(t, evt)
	evt.Timestamp = 0
	xevt.ID = uapi.LineEventRisingEdge
	assert.Equal(t, xevt, *evt)

	c.SetValue(1, 0)
	evt, err = readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	// test single line seqno paths
	c.SetValue(1, 1)
	evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
	require.Nil(t, err)
	require.NotNil(t, evt)
	evt.Timestamp = 0
	xevt.ID = uapi.LineEventRisingEdge
	xevt.Seqno++
	xevt.LineSeqno++
	assert.Equal(t, xevt, *evt)

	unix.Close(int(lr.Fd))

	if mockup.CheckKernelVersion(eventClockRealtimeKernel) != nil {
		return
	}

	// realtime timestamp
	lr.Lines = 1
	lr.Config.Flags |= uapi.LineFlagV2EventClockRealtime | uapi.LineFlagV2EdgeBoth
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)

	start := time.Now()
	c.SetValue(1, 0)
	evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
	end := time.Now()
	require.Nil(t, err)
	require.NotNil(t, evt)
	assert.LessOrEqual(t, uint64(start.UnixNano()), evt.Timestamp)
	assert.GreaterOrEqual(t, uint64(end.UnixNano()), evt.Timestamp)
	evt.Timestamp = 0
	xevt.ID = uapi.LineEventFallingEdge
	xevt.Seqno = 1
	xevt.LineSeqno = 1
	assert.Equal(t, xevt, *evt)

	start = time.Now()
	c.SetValue(1, 1)
	evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
	end = time.Now()
	require.Nil(t, err)
	require.NotNil(t, evt)
	assert.LessOrEqual(t, uint64(start.UnixNano()), evt.Timestamp)
	assert.GreaterOrEqual(t, uint64(end.UnixNano()), evt.Timestamp)
	evt.Timestamp = 0
	xevt.ID = uapi.LineEventRisingEdge
	xevt.Seqno++
	xevt.LineSeqno++
	assert.Equal(t, xevt, *evt)

	unix.Close(int(lr.Fd))
}

func readLineEventTimeout(fd int32, t time.Duration) (*uapi.LineEvent, error) {
	pollfd := unix.PollFd{Fd: int32(fd), Events: unix.POLLIN}
	n, err := unix.Poll([]unix.PollFd{pollfd}, int(t.Milliseconds()))
	if err != nil || n != 1 {
		return nil, err
	}
	evt, err := uapi.ReadLineEvent(uintptr(fd))
	if err != nil {
		return nil, err
	}
	return &evt, nil
}

func TestLineAttribute(t *testing.T) {
	var la uapi.LineAttribute

	la.Encode32(1, 1000000)
	assert.Equal(t, uapi.LineAttributeID(1), la.ID)
	assert.Zero(t, la.Padding)
	assert.Equal(t, uint32(1000000), la.Value32())

	la.Encode64(2, 200000000000)
	assert.Equal(t, uapi.LineAttributeID(2), la.ID)
	assert.Zero(t, la.Padding)
	assert.Equal(t, uint64(200000000000), la.Value64())
}

func TestLineFlagV2(t *testing.T) {
	var f uapi.LineFlagV2

	la := f.Encode()
	assert.Equal(t, uapi.LineAttributeIDFlags, la.ID)
	assert.Zero(t, la.Padding)
	assert.Equal(t, uint64(0), la.Value64())
	la.Encode64(uapi.LineAttributeIDFlags, 42000)
	f.Decode(la)
	assert.Equal(t, uapi.LineFlagV2(42000), f)

	f = uapi.LineFlagV2(1234567)
	la = f.Encode()
	assert.Equal(t, uapi.LineAttributeIDFlags, la.ID)
	assert.Zero(t, la.Padding)
	assert.Equal(t, uint64(1234567), la.Value64())
	f = 0
	f.Decode(la)
	assert.Equal(t, uapi.LineFlagV2(1234567), f)
}

func TestDebouncePeriod(t *testing.T) {
	var dp uapi.DebouncePeriod

	la := dp.Encode()
	assert.Equal(t, uapi.LineAttributeIDDebounce, la.ID)
	assert.Zero(t, la.Padding)
	assert.Equal(t, uint32(0), la.Value32())
	la.Encode32(uapi.LineAttributeIDDebounce, 42000)
	dp.Decode(la)
	assert.Equal(t, uapi.DebouncePeriod(42*time.Millisecond), dp)

	dp = uapi.DebouncePeriod(1234567 * time.Nanosecond)
	la = dp.Encode()
	assert.Equal(t, uapi.LineAttributeIDDebounce, la.ID)
	assert.Zero(t, la.Padding)
	assert.Equal(t, uint32(1234), la.Value32())
	dp = 0
	dp.Decode(la)
	assert.Equal(t, uapi.DebouncePeriod(1234*time.Microsecond), dp)
}

func TestOutputValues(t *testing.T) {
	var bits uapi.OutputValues

	la := bits.Encode()
	assert.Equal(t, uapi.LineAttributeIDOutputValues, la.ID)
	assert.Zero(t, la.Padding)
	assert.Equal(t, uint64(0), la.Value64())
	la.Encode64(uapi.LineAttributeIDOutputValues, 42234)
	bits.Decode(la)
	assert.Equal(t, uapi.OutputValues(42234), bits)

	bits = uapi.OutputValues(0x123456789)
	la = bits.Encode()
	assert.Equal(t, uapi.LineAttributeIDOutputValues, la.ID)
	assert.Zero(t, la.Padding)
	assert.Equal(t, uint64(0x123456789), la.Value64())
	bits = 0
	bits.Decode(la)
	assert.Equal(t, uapi.OutputValues(0x123456789), bits)
}

func TestNewLineBits(t *testing.T) {
	patterns := []struct {
		name string
		bits []int
		mask uapi.LineBitmap
	}{
		{
			"zero",
			[]int{0},
			1,
		},
		{
			"one",
			[]int{1},
			2,
		},
		{
			"three",
			[]int{3},
			8,
		},
		{
			"seven",
			[]int{0, 1, 2},
			7,
		},
		{
			"top",
			[]int{63},
			0x8000000000000000,
		},
		{
			"ends",
			[]int{0, 63},
			0x8000000000000001,
		},
		{
			"overflow",
			[]int{64},
			0,
		},
	}
	for _, p := range patterns {
		tf := func(t *testing.T) {
			mask := uapi.NewLineBits(p.bits...)
			assert.Equal(t, p.mask, mask)
		}
		t.Run(p.name, tf)
	}
}

func TestNewLineBitmap(t *testing.T) {
	patterns := []struct {
		name string
		bits []int
		mask uapi.LineBitmap
	}{
		{
			"zero",
			[]int{0},
			0,
		},
		{
			"one",
			[]int{1},
			1,
		},
		{
			"three",
			[]int{1, 1},
			3,
		},
		{
			"seven",
			[]int{1, 1, 1},
			7,
		},
		{
			"max",
			[]int{
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
			},
			0xffffffffffffffff,
		},
		{
			"overflow",
			[]int{
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
			},
			0xffffffffffffffff,
		},
	}
	for _, p := range patterns {
		tf := func(t *testing.T) {
			mask := uapi.NewLineBitmap(p.bits...)
			assert.Equal(t, p.mask, mask)
		}
		t.Run(p.name, tf)
	}
}

func TestNewLineBitMask(t *testing.T) {
	patterns := []struct {
		name string
		bits int
		mask uapi.LineBitmap
	}{
		{
			"zero",
			0,
			0,
		},
		{
			"one",
			1,
			1,
		},
		{
			"two",
			2,
			3,
		},
		{
			"three",
			3,
			7,
		},
		{
			"max",
			64,
			0xffffffffffffffff,
		},
		{
			"overflow",
			65,
			0xffffffffffffffff,
		},
	}
	for _, p := range patterns {
		tf := func(t *testing.T) {
			mask := uapi.NewLineBitMask(p.bits)
			assert.Equal(t, p.mask, mask)
		}
		t.Run(p.name, tf)
	}
}

func TestLineBitmap(t *testing.T) {
	lb := uapi.LineBitmap(0)

	assert.Zero(t, lb.Get(0))
	lb = lb.Set(2, 1)
	assert.Equal(t, uapi.LineBitmap(4), lb)
	assert.Equal(t, 0, lb.Get(0))
	assert.Equal(t, 0, lb.Get(1))
	assert.Equal(t, 1, lb.Get(2))

	lb = lb.Set(0, 1)
	assert.Equal(t, uapi.LineBitmap(5), lb)
	assert.Equal(t, 1, lb.Get(0))
	assert.Equal(t, 0, lb.Get(1))
	assert.Equal(t, 1, lb.Get(2))

	lb = lb.Set(0, 0)
	assert.Equal(t, uapi.LineBitmap(4), lb)
	assert.Equal(t, 0, lb.Get(0))
	assert.Equal(t, 0, lb.Get(1))
	assert.Equal(t, 1, lb.Get(2))

	lb = lb.Set(2, 0)
	assert.Zero(t, lb.Get(0))
}

func TestLineConfig(t *testing.T) {
	var lc uapi.LineConfig

	// remove when empty
	assert.Zero(t, lc.NumAttrs)
	lc.RemoveAttributeID(1)
	assert.Zero(t, lc.NumAttrs)
	lc.RemoveAttributeID(0)
	assert.Zero(t, lc.NumAttrs)

	// add
	lca := uapi.LineConfigAttribute{
		Attr: uapi.LineAttribute{
			ID: 56,
		},
		Mask: uapi.NewLineBitMask(63),
	}
	lca2 := uapi.LineConfigAttribute{
		Attr: uapi.LineAttribute{
			ID: 23,
		},
		Mask: uapi.NewLineBitMask(64),
	}

	lc.AddAttribute(lca)
	assert.Equal(t, uint32(1), lc.NumAttrs)
	lc.AddAttribute(lca2)
	assert.Equal(t, uint32(2), lc.NumAttrs)
	lc.AddAttribute(lca)
	assert.Equal(t, uint32(3), lc.NumAttrs)
	assert.Equal(t, lca, lc.Attrs[0])
	assert.Equal(t, lca2, lc.Attrs[1])
	assert.Equal(t, lca, lc.Attrs[2])

	// remove by id
	lc.RemoveAttributeID(42)
	assert.Equal(t, uint32(3), lc.NumAttrs)
	lc.RemoveAttributeID(56)
	assert.Equal(t, uint32(1), lc.NumAttrs)

	// remove by value
	lc.AddAttribute(lca)
	lc.AddAttribute(lca)
	assert.Equal(t, uint32(3), lc.NumAttrs)
	lc.RemoveAttribute(lca2)
	assert.Equal(t, uint32(2), lc.NumAttrs)
	lc.RemoveAttribute(lca)
	assert.Zero(t, lc.NumAttrs)
}

func TestDebounce(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	requireMockup(t)
	c, err := mock.Chip(0)
	require.Nil(t, err)
	f, err := os.Open(c.DevPath)
	require.Nil(t, err)
	defer f.Close()
	err = c.SetValue(1, 0)
	require.Nil(t, err)
	lr := uapi.LineRequest{
		Lines:   1,
		Offsets: [uapi.LinesMax]uint32{1},
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagV2Input |
				uapi.LineFlagV2EdgeBoth,
			NumAttrs: 1,
		},
	}
	lr.Config.Attrs[0].Mask = 1
	lr.Config.Attrs[0].Attr = uapi.DebouncePeriod(debouncePeriod).Encode()
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)

	evt, err := readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	// toggle faster than the debounce period - should be filtered
	for i := 0; i < 10; i++ {
		c.SetValue(1, 1)
		time.Sleep(clkTick)
		checkLineValue(t, lr, 0)
		c.SetValue(1, 0)
		time.Sleep(clkTick)
		checkLineValue(t, lr, 0)
	}
	// but this change will persist and get through...
	c.SetValue(1, 1)

	evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
	assert.Nil(t, err)
	require.NotNil(t, evt)
	assert.Equal(t, uint32(1), evt.Offset)
	assert.Equal(t, uapi.LineEventRisingEdge, evt.ID)
	lastTime := evt.Timestamp

	checkLineValue(t, lr, 1)

	evt, err = readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	// toggle slower than the debounce period - should all get through
	for i := 0; i < 2; i++ {
		c.SetValue(1, 0)
		time.Sleep(2 * debouncePeriod)
		checkLineValue(t, lr, 0)
		c.SetValue(1, 1)
		time.Sleep(2 * debouncePeriod)
		checkLineValue(t, lr, 1)
	}
	c.SetValue(1, 0)
	time.Sleep(2 * debouncePeriod)
	checkLineValue(t, lr, 0)
	for i := 0; i < 2; i++ {
		evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
		assert.Nil(t, err)
		require.NotNil(t, evt)
		assert.Equal(t, uint32(1), evt.Offset)
		assert.Equal(t, uapi.LineEventFallingEdge, evt.ID)
		assert.GreaterOrEqual(t, evt.Timestamp-lastTime, uint64(10000000))
		lastTime = evt.Timestamp

		evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
		assert.Nil(t, err)
		require.NotNil(t, evt)
		assert.Equal(t, uint32(1), evt.Offset)
		assert.Equal(t, uapi.LineEventRisingEdge, evt.ID)
		assert.GreaterOrEqual(t, evt.Timestamp-lastTime, uint64(10000000))
		lastTime = evt.Timestamp
	}
	evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
	assert.Nil(t, err)
	require.NotNil(t, evt)
	assert.Equal(t, uint32(1), evt.Offset)
	assert.Equal(t, uapi.LineEventFallingEdge, evt.ID)
	assert.GreaterOrEqual(t, evt.Timestamp-lastTime, uint64(10000000))
	evt, err = readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	unix.Close(int(lr.Fd))
}

func checkLineValue(t *testing.T, lr uapi.LineRequest, v int) {
	t.Helper()
	lv := uapi.LineValues{Mask: 1}
	err := uapi.GetLineValuesV2(uintptr(lr.Fd), &lv)
	assert.Nil(t, err)
	assert.Equal(t, v, lv.Get(0))
}

func readLineInfoChangedV2Timeout(fd uintptr,
	t time.Duration) (*uapi.LineInfoChangedV2, error) {

	pollfd := unix.PollFd{Fd: int32(fd), Events: unix.POLLIN}
	n, err := unix.Poll([]unix.PollFd{pollfd}, int(t.Milliseconds()))
	if err != nil || n != 1 {
		return nil, err
	}
	infoChanged, err := uapi.ReadLineInfoChangedV2(fd)
	if err != nil {
		return nil, err
	}
	return &infoChanged, nil
}

func TestLineFlagsV2(t *testing.T) {
	assert.True(t, uapi.LineFlagV2(0).IsAvailable())
	assert.False(t, uapi.LineFlagV2(0).IsUsed())
	assert.False(t, uapi.LineFlagV2(0).IsActiveLow())
	assert.False(t, uapi.LineFlagV2(0).IsInput())
	assert.False(t, uapi.LineFlagV2(0).IsOutput())
	assert.False(t, uapi.LineFlagV2(0).IsRisingEdge())
	assert.False(t, uapi.LineFlagV2(0).IsFallingEdge())
	assert.False(t, uapi.LineFlagV2(0).IsBothEdges())
	assert.False(t, uapi.LineFlagV2(0).IsOpenDrain())
	assert.False(t, uapi.LineFlagV2(0).IsOpenSource())
	assert.False(t, uapi.LineFlagV2(0).IsBiasDisabled())
	assert.False(t, uapi.LineFlagV2(0).IsBiasPullUp())
	assert.False(t, uapi.LineFlagV2(0).IsBiasPullDown())
	assert.False(t, uapi.LineFlagV2(0).HasRealtimeEventClock())
	assert.False(t, uapi.LineFlagV2Used.IsAvailable())
	assert.True(t, uapi.LineFlagV2Used.IsUsed())
	assert.True(t, uapi.LineFlagV2ActiveLow.IsActiveLow())
	assert.True(t, uapi.LineFlagV2Input.IsInput())
	assert.True(t, uapi.LineFlagV2Output.IsOutput())
	assert.True(t, uapi.LineFlagV2EdgeRising.IsRisingEdge())
	assert.True(t, uapi.LineFlagV2EdgeFalling.IsFallingEdge())
	assert.True(t, uapi.LineFlagV2EdgeBoth.IsBothEdges())
	assert.False(t, uapi.LineFlagV2EdgeRising.IsBothEdges())
	assert.False(t, uapi.LineFlagV2EdgeFalling.IsBothEdges())
	assert.True(t, uapi.LineFlagV2BiasDisabled.IsBiasDisabled())
	assert.True(t, uapi.LineFlagV2BiasPullUp.IsBiasPullUp())
	assert.True(t, uapi.LineFlagV2BiasPullDown.IsBiasPullDown())
	assert.True(t, uapi.LineFlagV2EventClockRealtime.HasRealtimeEventClock())
}
