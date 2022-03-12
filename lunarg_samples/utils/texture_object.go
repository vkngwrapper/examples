package utils

import (
	"github.com/CannibalVox/VKng/core/common"
	"github.com/CannibalVox/VKng/core/core1_0"
)

type TextureObject struct {
	Sampler core1_0.Sampler

	Image       core1_0.Image
	ImageLayout common.ImageLayout

	NeedsStaging bool
	Buffer       core1_0.Buffer
	BufferSize   int

	ImageMemory         core1_0.DeviceMemory
	BufferMemory        core1_0.DeviceMemory
	View                core1_0.ImageView
	TexWidth, TexHeight int
}
