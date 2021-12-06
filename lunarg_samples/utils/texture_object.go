package utils

import (
	"github.com/CannibalVox/VKng/core"
	"github.com/CannibalVox/VKng/core/common"
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
	View                core.ImageView
	TexWidth, TexHeight int
}
