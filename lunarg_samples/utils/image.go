package utils

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/vkngwrapper/core/v3/core1_0"
	"github.com/vkngwrapper/extensions/v3/khr_swapchain"
)

func (i *SampleInfo) SetImageLayout(image core1_0.Image, aspectMask core1_0.ImageAspectFlags, oldImageLayout core1_0.ImageLayout, newImageLayout core1_0.ImageLayout, sourceStages core1_0.PipelineStageFlags, destStages core1_0.PipelineStageFlags) error {
	imageBarrierOptions := core1_0.ImageMemoryBarrier{
		OldLayout:           oldImageLayout,
		NewLayout:           newImageLayout,
		SrcQueueFamilyIndex: -1,
		DstQueueFamilyIndex: -1,
		Image:               image,
		SubresourceRange: core1_0.ImageSubresourceRange{
			AspectMask:     aspectMask,
			BaseMipLevel:   0,
			LevelCount:     1,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
	}

	switch oldImageLayout {
	case core1_0.ImageLayoutColorAttachmentOptimal:
		imageBarrierOptions.SrcAccessMask = core1_0.AccessColorAttachmentWrite
	case core1_0.ImageLayoutTransferDstOptimal:
		imageBarrierOptions.SrcAccessMask = core1_0.AccessTransferWrite
	case core1_0.ImageLayoutPreInitialized:
		imageBarrierOptions.SrcAccessMask = core1_0.AccessHostWrite
	}

	switch newImageLayout {
	case core1_0.ImageLayoutTransferDstOptimal:
		imageBarrierOptions.DstAccessMask = core1_0.AccessTransferWrite
	case core1_0.ImageLayoutTransferSrcOptimal:
		imageBarrierOptions.DstAccessMask = core1_0.AccessTransferRead
	case core1_0.ImageLayoutShaderReadOnlyOptimal:
		imageBarrierOptions.DstAccessMask = core1_0.AccessShaderRead
	case core1_0.ImageLayoutColorAttachmentOptimal:
		imageBarrierOptions.DstAccessMask = core1_0.AccessColorAttachmentWrite
	case core1_0.ImageLayoutDepthStencilAttachmentOptimal:
		imageBarrierOptions.DstAccessMask = core1_0.AccessDepthStencilAttachmentWrite
	}

	return i.DeviceDriver.CmdPipelineBarrier(i.Cmd, sourceStages, destStages, 0, nil, nil, []core1_0.ImageMemoryBarrier{imageBarrierOptions})
}

func (i *SampleInfo) WritePNG(baseName string) error {
	mappableImage, _, err := i.DeviceDriver.CreateImage(nil, core1_0.ImageCreateInfo{
		ImageType: core1_0.ImageType2D,
		Format:    i.Format,
		Extent: core1_0.Extent3D{
			Width:  i.Width,
			Height: i.Height,
			Depth:  1,
		},
		MipLevels:     1,
		ArrayLayers:   1,
		Samples:       core1_0.Samples1,
		Tiling:        core1_0.ImageTilingLinear,
		Usage:         core1_0.ImageUsageTransferDst,
		SharingMode:   core1_0.SharingModeExclusive,
		InitialLayout: core1_0.ImageLayoutUndefined,
	})
	if err != nil {
		return err
	}

	memReqs := i.DeviceDriver.GetImageMemoryRequirements(mappableImage)
	memoryTypeIndex, err := i.MemoryTypeFromProperties(memReqs.MemoryTypeBits, core1_0.MemoryPropertyHostVisible|core1_0.MemoryPropertyHostCoherent)
	if err != nil {
		return err
	}

	mappableMemory, _, err := i.DeviceDriver.AllocateMemory(nil, core1_0.MemoryAllocateInfo{
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memoryTypeIndex,
	})
	if err != nil {
		return err
	}

	_, err = i.DeviceDriver.BindImageMemory(mappableImage, mappableMemory, 0)
	if err != nil {
		return err
	}

	_, err = i.DeviceDriver.BeginCommandBuffer(i.Cmd, core1_0.CommandBufferBeginInfo{})
	if err != nil {
		return err
	}

	err = i.SetImageLayout(mappableImage, core1_0.ImageAspectColor, core1_0.ImageLayoutUndefined, core1_0.ImageLayoutTransferDstOptimal, core1_0.PipelineStageTopOfPipe, core1_0.PipelineStageTransfer)
	if err != nil {
		return err
	}

	err = i.SetImageLayout(i.Buffers[i.CurrentBuffer].Image, core1_0.ImageAspectColor, khr_swapchain.ImageLayoutPresentSrc, core1_0.ImageLayoutTransferSrcOptimal, core1_0.PipelineStageBottomOfPipe, core1_0.PipelineStageTransfer)
	if err != nil {
		return err
	}

	err = i.DeviceDriver.CmdCopyImage(i.Cmd, i.Buffers[i.CurrentBuffer].Image,
		core1_0.ImageLayoutTransferSrcOptimal,
		mappableImage,
		core1_0.ImageLayoutTransferDstOptimal,
		core1_0.ImageCopy{
			SrcSubresource: core1_0.ImageSubresourceLayers{
				AspectMask:     core1_0.ImageAspectColor,
				MipLevel:       0,
				BaseArrayLayer: 0,
				LayerCount:     1,
			},
			DstSubresource: core1_0.ImageSubresourceLayers{
				AspectMask:     core1_0.ImageAspectColor,
				MipLevel:       0,
				BaseArrayLayer: 0,
				LayerCount:     1,
			},
			SrcOffset: core1_0.Offset3D{0, 0, 0},
			DstOffset: core1_0.Offset3D{0, 0, 0},
			Extent:    core1_0.Extent3D{i.Width, i.Height, 1},
		},
	)
	if err != nil {
		return err
	}

	err = i.SetImageLayout(mappableImage, core1_0.ImageAspectColor, core1_0.ImageLayoutTransferDstOptimal, core1_0.ImageLayoutGeneral, core1_0.PipelineStageTransfer, core1_0.PipelineStageHost)
	if err != nil {
		return err
	}

	_, err = i.DeviceDriver.EndCommandBuffer(i.Cmd)
	if err != nil {
		return err
	}

	cmdFence, _, err := i.DeviceDriver.CreateFence(nil, core1_0.FenceCreateInfo{})
	if err != nil {
		return err
	}

	_, err = i.DeviceDriver.QueueSubmit(i.GraphicsQueue, &cmdFence,
		core1_0.SubmitInfo{
			CommandBuffers: []core1_0.CommandBuffer{i.Cmd},
		},
	)
	if err != nil {
		return err
	}

	for {
		res, err := i.DeviceDriver.WaitForFences(true, FenceTimeout, cmdFence)
		if err != nil {
			return err
		}

		if res != core1_0.VKTimeout {
			break
		}
	}

	i.DeviceDriver.DestroyFence(cmdFence, nil)

	filename := fmt.Sprintf("%s.png", baseName)
	subresourceLayout := i.DeviceDriver.GetImageSubresourceLayout(mappableImage, &core1_0.ImageSubresource{
		AspectMask: core1_0.ImageAspectColor,
		MipLevel:   0,
		ArrayLayer: 0,
	})

	memPtr, _, err := i.DeviceDriver.MapMemory(mappableMemory, 0, memReqs.Size, 0)
	if err != nil {
		return err
	}

	dataBuffer := unsafe.Slice((*byte)(memPtr), memReqs.Size)
	bufferIndex := subresourceLayout.Offset

	outImg := image.NewRGBA(image.Rectangle{
		Min: image.Point{X: 0, Y: 0},
		Max: image.Point{X: i.Width, Y: i.Height},
	})

	for y := 0; y < i.Height; y++ {
		rowIndex := bufferIndex
		if i.Format == core1_0.FormatB8G8R8A8UnsignedNormalized || i.Format == core1_0.FormatB8G8R8A8SRGB {
			for x := 0; x < i.Width; x++ {
				outImg.Set(x, y, color.RGBA{
					B: dataBuffer[rowIndex],
					G: dataBuffer[rowIndex+1],
					R: dataBuffer[rowIndex+2],
					A: dataBuffer[rowIndex+3],
				})
				rowIndex += 4
			}
		} else if i.Format == core1_0.FormatR8G8B8A8UnsignedNormalized {
			for x := 0; x < i.Width; x++ {
				outImg.Set(x, y, color.RGBA{
					R: dataBuffer[rowIndex],
					G: dataBuffer[rowIndex+1],
					B: dataBuffer[rowIndex+2],
					A: dataBuffer[rowIndex+3],
				})
				rowIndex += 4
			}
		} else {
			return errors.New("unrecognized image format - will not write image files")
		}
		bufferIndex += subresourceLayout.RowPitch
	}

	i.DeviceDriver.UnmapMemory(mappableMemory)
	i.DeviceDriver.DestroyImage(mappableImage, nil)
	i.DeviceDriver.FreeMemory(mappableMemory, nil)

	writeFile, err := os.Create(filename)
	if err != nil {
		return err
	}

	return png.Encode(writeFile, outImg)
}

func (i *SampleInfo) InitTextureBuffer(textureObj *TextureObject) error {
	var err error
	textureObj.Buffer, _, err = i.DeviceDriver.CreateBuffer(nil, core1_0.BufferCreateInfo{
		Size:        textureObj.TexWidth * textureObj.TexHeight * 4,
		Usage:       core1_0.BufferUsageTransferSrc,
		SharingMode: core1_0.SharingModeExclusive,
	})
	if err != nil {
		return err
	}

	memReqs := i.DeviceDriver.GetBufferMemoryRequirements(textureObj.Buffer)
	textureObj.BufferSize = memReqs.Size

	requirements := core1_0.MemoryPropertyHostVisible | core1_0.MemoryPropertyHostCoherent
	memoryIndex, err := i.MemoryTypeFromProperties(memReqs.MemoryTypeBits, requirements)
	if err != nil {
		return err
	}

	/* allocate memory */
	textureObj.BufferMemory, _, err = i.DeviceDriver.AllocateMemory(nil, core1_0.MemoryAllocateInfo{
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memoryIndex,
	})
	if err != nil {
		return err
	}

	_, err = i.DeviceDriver.BindBufferMemory(textureObj.Buffer, textureObj.BufferMemory, 0)
	return err
}

func (i *SampleInfo) InitImage(textureReader io.Reader, extraUsages core1_0.ImageUsageFlags, extraFeatures core1_0.FormatFeatureFlags) (*TextureObject, error) {
	image, _, err := image.Decode(textureReader)
	if err != nil {
		return nil, err
	}

	textureObj := &TextureObject{}
	textureObj.TexWidth = image.Bounds().Size().X
	textureObj.TexHeight = image.Bounds().Size().Y

	formatProps := i.InstanceDriver.GetPhysicalDeviceFormatProperties(i.Gpus[0], core1_0.FormatR8G8B8A8UnsignedNormalized)

	/* See if we can use a linear tiled image for a texture, if not, we will
	 * need a staging buffer for the texture data */
	allFeatures := core1_0.FormatFeatureSampledImage | extraFeatures
	textureObj.NeedsStaging = (formatProps.LinearTilingFeatures & allFeatures) != allFeatures

	if textureObj.NeedsStaging {
		if (formatProps.OptimalTilingFeatures & allFeatures) != allFeatures {
			return nil, errors.Errorf("Format %s cannot support featureset %s\n", core1_0.FormatR8G8B8A8UnsignedNormalized, allFeatures)
		}
		err = i.InitTextureBuffer(textureObj)
		if err != nil {
			return nil, err
		}
		extraUsages |= core1_0.ImageUsageTransferDst
	}

	imageOptions := core1_0.ImageCreateInfo{
		ImageType:   core1_0.ImageType2D,
		Format:      core1_0.FormatR8G8B8A8UnsignedNormalized,
		Extent:      core1_0.Extent3D{Width: textureObj.TexWidth, Height: textureObj.TexHeight, Depth: 1},
		MipLevels:   1,
		ArrayLayers: 1,
		Samples:     NumSamples,
		Usage:       core1_0.ImageUsageSampled | extraUsages,
		SharingMode: core1_0.SharingModeExclusive,
	}
	if textureObj.NeedsStaging {
		imageOptions.Tiling = core1_0.ImageTilingOptimal
		imageOptions.InitialLayout = core1_0.ImageLayoutUndefined
	} else {
		imageOptions.Tiling = core1_0.ImageTilingLinear
		imageOptions.InitialLayout = core1_0.ImageLayoutPreInitialized
	}

	textureObj.Image, _, err = i.DeviceDriver.CreateImage(nil, imageOptions)
	if err != nil {
		return nil, err
	}

	memReqs := i.DeviceDriver.GetImageMemoryRequirements(textureObj.Image)

	var requirements core1_0.MemoryPropertyFlags
	if !textureObj.NeedsStaging {
		requirements = core1_0.MemoryPropertyHostVisible | core1_0.MemoryPropertyHostCoherent
	}

	memoryIndex, err := i.MemoryTypeFromProperties(memReqs.MemoryTypeBits, requirements)
	if err != nil {
		return nil, err
	}

	/* allocate memory */
	textureObj.ImageMemory, _, err = i.DeviceDriver.AllocateMemory(nil, core1_0.MemoryAllocateInfo{
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memoryIndex,
	})
	if err != nil {
		return nil, err
	}

	/* bind memory */
	_, err = i.DeviceDriver.BindImageMemory(textureObj.Image, textureObj.ImageMemory, 0)
	if err != nil {
		return nil, err
	}

	_, err = i.DeviceDriver.EndCommandBuffer(i.Cmd)
	if err != nil {
		return nil, err
	}

	cmdFence, _, err := i.DeviceDriver.CreateFence(nil, core1_0.FenceCreateInfo{})
	if err != nil {
		return nil, err
	}

	/* Queue the command buffer for execution */
	_, err = i.DeviceDriver.QueueSubmit(i.GraphicsQueue, &cmdFence,
		core1_0.SubmitInfo{
			CommandBuffers: []core1_0.CommandBuffer{i.Cmd},
		},
	)
	if err != nil {
		return nil, err
	}

	subResource := &core1_0.ImageSubresource{
		AspectMask: core1_0.ImageAspectColor,
		MipLevel:   0,
		ArrayLayer: 0,
	}
	layout := &core1_0.SubresourceLayout{}
	if !textureObj.NeedsStaging {
		/* Get the subresource layout so we know what the row pitch is */
		layout = i.DeviceDriver.GetImageSubresourceLayout(textureObj.Image, subResource)
	}

	/* Make sure command buffer is finished before mapping */
	for {
		res, err := i.DeviceDriver.WaitForFences(true, FenceTimeout, cmdFence)
		if err != nil {
			return nil, err
		}

		if res != core1_0.VKTimeout {
			break
		}
	}

	i.DeviceDriver.DestroyFence(cmdFence, nil)

	var dataPtr unsafe.Pointer
	var data []byte
	if textureObj.NeedsStaging {
		dataPtr, _, err = i.DeviceDriver.MapMemory(textureObj.BufferMemory, 0, textureObj.BufferSize, 0)
		data = ([]byte)(unsafe.Slice((*byte)(dataPtr), textureObj.BufferSize))
	} else {
		dataPtr, _, err = i.DeviceDriver.MapMemory(textureObj.ImageMemory, 0, memReqs.Size, 0)
		data = ([]byte)(unsafe.Slice((*byte)(dataPtr), memReqs.Size))
	}
	if err != nil {
		return nil, err
	}

	/* Read the image file into the mappable image's memory */
	var dataIndex = 0
	for y := image.Bounds().Min.Y; y < image.Bounds().Max.Y; y++ {
		rowIndex := dataIndex
		for x := image.Bounds().Min.X; x < image.Bounds().Max.Y; x++ {
			r, g, b, a := image.At(x, y).RGBA()
			data[rowIndex] = byte(r)
			data[rowIndex+1] = byte(g)
			data[rowIndex+2] = byte(b)
			data[rowIndex+3] = byte(a)
			rowIndex += 4
		}
		if textureObj.NeedsStaging {
			dataIndex += textureObj.TexWidth * 4
		} else {
			dataIndex += layout.RowPitch
		}
	}

	if textureObj.NeedsStaging {
		i.DeviceDriver.UnmapMemory(textureObj.BufferMemory)
	} else {
		i.DeviceDriver.UnmapMemory(textureObj.ImageMemory)
	}

	_, err = i.DeviceDriver.ResetCommandBuffer(i.Cmd, 0)
	if err != nil {
		return nil, err
	}
	_, err = i.DeviceDriver.BeginCommandBuffer(i.Cmd, core1_0.CommandBufferBeginInfo{})
	if err != nil {
		return nil, err
	}

	if !textureObj.NeedsStaging {
		/* If we can use the linear tiled image as a texture, just do it */
		textureObj.ImageLayout = core1_0.ImageLayoutShaderReadOnlyOptimal
		err = i.SetImageLayout(textureObj.Image, core1_0.ImageAspectColor, core1_0.ImageLayoutPreInitialized, textureObj.ImageLayout, core1_0.PipelineStageHost, core1_0.PipelineStageFragmentShader)
		if err != nil {
			return nil, err
		}
	} else {
		/* Since we're going to blit to the texture image, set its layout to
		 * DESTINATION_OPTIMAL */
		err = i.SetImageLayout(textureObj.Image, core1_0.ImageAspectColor, core1_0.ImageLayoutUndefined, core1_0.ImageLayoutTransferDstOptimal, core1_0.PipelineStageTopOfPipe, core1_0.PipelineStageTransfer)
		if err != nil {
			return nil, err
		}

		/* Put the copy command into the command buffer */
		err = i.DeviceDriver.CmdCopyBufferToImage(i.Cmd, textureObj.Buffer, textureObj.Image, core1_0.ImageLayoutTransferDstOptimal,
			core1_0.BufferImageCopy{
				BufferOffset:      0,
				BufferRowLength:   textureObj.TexWidth,
				BufferImageHeight: textureObj.TexHeight,
				ImageSubresource: core1_0.ImageSubresourceLayers{
					AspectMask:     core1_0.ImageAspectColor,
					MipLevel:       0,
					BaseArrayLayer: 0,
					LayerCount:     1,
				},
				ImageOffset: core1_0.Offset3D{0, 0, 0},
				ImageExtent: core1_0.Extent3D{textureObj.TexWidth, textureObj.TexHeight, 1},
			},
		)
		if err != nil {
			return nil, err
		}

		/* Set the layout for the texture image from DESTINATION_OPTIMAL to
		 * SHADER_READ_ONLY */
		textureObj.ImageLayout = core1_0.ImageLayoutShaderReadOnlyOptimal
		err = i.SetImageLayout(textureObj.Image, core1_0.ImageAspectColor, core1_0.ImageLayoutTransferDstOptimal, textureObj.ImageLayout, core1_0.PipelineStageTransfer, core1_0.PipelineStageFragmentShader)
		if err != nil {
			return nil, err
		}
	}

	/* create image view */
	textureObj.View, _, err = i.DeviceDriver.CreateImageView(nil, core1_0.ImageViewCreateInfo{
		Image:    textureObj.Image,
		ViewType: core1_0.ImageViewType2D,
		Format:   core1_0.FormatR8G8B8A8UnsignedNormalized,
		Components: core1_0.ComponentMapping{
			R: core1_0.ComponentSwizzleRed,
			G: core1_0.ComponentSwizzleGreen,
			B: core1_0.ComponentSwizzleBlue,
			A: core1_0.ComponentSwizzleAlpha,
		},
		SubresourceRange: core1_0.ImageSubresourceRange{
			AspectMask:     core1_0.ImageAspectColor,
			BaseMipLevel:   0,
			LevelCount:     1,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
	})
	return textureObj, err
}

func (i *SampleInfo) InitTexture(textureReader io.Reader, extraUsages core1_0.ImageUsageFlags, extraFeatures core1_0.FormatFeatureFlags) error {
	/* create image */
	texObj, err := i.InitImage(textureReader, extraUsages, extraFeatures)
	if err != nil {
		return err
	}

	/* create sampler */
	texObj.Sampler, err = i.InitSampler()
	if err != nil {
		return err
	}

	i.Textures = append(i.Textures, texObj)

	/* track a description of the texture */
	i.TextureData.ImageInfo.ImageView = texObj.View
	i.TextureData.ImageInfo.Sampler = texObj.Sampler
	i.TextureData.ImageInfo.ImageLayout = core1_0.ImageLayoutShaderReadOnlyOptimal

	return nil
}

func (i *SampleInfo) DestroyTextures() {
	for ind := 0; ind < len(i.Textures); ind++ {
		i.DeviceDriver.DestroySampler(i.Textures[ind].Sampler, nil)
		i.DeviceDriver.DestroyImageView(i.Textures[ind].View, nil)
		i.DeviceDriver.DestroyImage(i.Textures[ind].Image, nil)
		i.DeviceDriver.FreeMemory(i.Textures[ind].ImageMemory, nil)

		if i.Textures[ind].Buffer.Initialized() {
			i.DeviceDriver.DestroyBuffer(i.Textures[ind].Buffer, nil)
		}

		if i.Textures[ind].BufferMemory.Initialized() {
			i.DeviceDriver.FreeMemory(i.Textures[ind].BufferMemory, nil)
		}
	}
}
