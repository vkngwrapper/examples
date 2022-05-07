package utils

import (
	"github.com/CannibalVox/VKng/core"
	"github.com/CannibalVox/VKng/core/common"
	"github.com/CannibalVox/VKng/core/core1_0"
)

type TextureObject struct {
	Sampler core.Sampler

	Image       core.Image
	ImageLayout common.ImageLayout

	NeedsStaging bool
	Buffer       core.Buffer
	BufferSize   int

	ImageMemory         core.DeviceMemory
	BufferMemory        core.DeviceMemory
	View                core1_0.ImageView
	TexWidth, TexHeight int
}
