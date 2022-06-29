package utils

import (
	"github.com/CannibalVox/VKng/core/core1_0"
	"time"
)

const (
	PreferredSurfaceFormat core1_0.DataFormat   = core1_0.DataFormatB8G8R8A8UnsignedNormalized
	NumSamples             core1_0.SampleCounts = core1_0.Samples1

	FenceTimeout      = 100 * time.Millisecond
	NumDescriptorSets = 1
)
