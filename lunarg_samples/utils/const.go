package utils

import (
	"github.com/vkngwrapper/core/v2/core1_0"
	"time"
)

const (
	PreferredSurfaceFormat core1_0.Format           = core1_0.FormatB8G8R8A8UnsignedNormalized
	NumSamples             core1_0.SampleCountFlags = core1_0.Samples1

	FenceTimeout      = 100 * time.Millisecond
	NumDescriptorSets = 1
)
