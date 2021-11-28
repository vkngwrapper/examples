package utils

import (
	"github.com/CannibalVox/VKng/core/common"
	"time"
)

const (
	PreferredSurfaceFormat common.DataFormat   = common.FormatB8G8R8A8UnsignedNormalized
	NumSamples             common.SampleCounts = common.Samples1

	FenceTimeout = 100 * time.Millisecond
)
