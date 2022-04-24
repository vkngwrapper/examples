package utils

import (
	"fmt"
	"github.com/CannibalVox/VKng/core/common"
	"github.com/CannibalVox/VKng/core/core1_0"
	"github.com/CannibalVox/VKng/extensions/khr_swapchain"
	"github.com/cockroachdb/errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"unsafe"
)

func (i *SampleInfo) SetImageLayout(image core1_0.Image, aspectMask common.ImageAspectFlags, oldImageLayout common.ImageLayout, newImageLayout common.ImageLayout, sourceStages common.PipelineStages, destStages common.PipelineStages) error {
	imageBarrierOptions := core1_0.ImageMemoryBarrierOptions{
		OldLayout:           oldImageLayout,
		NewLayout:           newImageLayout,
		SrcQueueFamilyIndex: -1,
		DstQueueFamilyIndex: -1,
		Image:               image,
		SubresourceRange: common.ImageSubresourceRange{
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

	return i.Cmd.CmdPipelineBarrier(sourceStages, destStages, 0, nil, nil, []core1_0.ImageMemoryBarrierOptions{imageBarrierOptions})
}

func (i *SampleInfo) WritePNG(baseName string) error {
	mappableImage, _, err := i.Loader.CreateImage(i.Device, nil, core1_0.ImageOptions{
		ImageType: core1_0.ImageType2D,
		Format:    i.Format,
		Extent: common.Extent3D{
			Width:  i.Width,
			Height: i.Height,
			Depth:  1,
		},
		MipLevels:     1,
		ArrayLayers:   1,
		Samples:       core1_0.Samples1,
		Tiling:        core1_0.ImageTilingLinear,
		Usage:         core1_0.ImageUsageTransferDst,
		SharingMode:   core1_0.SharingExclusive,
		InitialLayout: core1_0.ImageLayoutUndefined,
	})
	if err != nil {
		return err
	}

	memReqs := mappableImage.MemoryRequirements()
	memoryTypeIndex, err := i.MemoryTypeFromProperties(memReqs.MemoryType, core1_0.MemoryPropertyHostVisible|core1_0.MemoryPropertyHostCoherent)
	if err != nil {
		return err
	}

	mappableMemory, _, err := i.Loader.AllocateMemory(i.Device, nil, core1_0.DeviceMemoryOptions{
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memoryTypeIndex,
	})
	if err != nil {
		return err
	}

	_, err = mappableImage.BindImageMemory(mappableMemory, 0)
	if err != nil {
		return err
	}

	_, err = i.Cmd.Begin(core1_0.BeginOptions{})
	if err != nil {
		return err
	}

	err = i.SetImageLayout(mappableImage, core1_0.AspectColor, core1_0.ImageLayoutUndefined, core1_0.ImageLayoutTransferDstOptimal, core1_0.PipelineStageTopOfPipe, core1_0.PipelineStageTransfer)
	if err != nil {
		return err
	}

	err = i.SetImageLayout(i.Buffers[i.CurrentBuffer].Image, core1_0.AspectColor, khr_swapchain.ImageLayoutPresentSrc, core1_0.ImageLayoutTransferSrcOptimal, core1_0.PipelineStageBottomOfPipe, core1_0.PipelineStageTransfer)
	if err != nil {
		return err
	}

	err = i.Cmd.CmdCopyImage(i.Buffers[i.CurrentBuffer].Image,
		core1_0.ImageLayoutTransferSrcOptimal,
		mappableImage,
		core1_0.ImageLayoutTransferDstOptimal,
		[]core1_0.ImageCopy{
			{
				SrcSubresource: common.ImageSubresourceLayers{
					AspectMask:     core1_0.AspectColor,
					MipLevel:       0,
					BaseArrayLayer: 0,
					LayerCount:     1,
				},
				DstSubresource: common.ImageSubresourceLayers{
					AspectMask:     core1_0.AspectColor,
					MipLevel:       0,
					BaseArrayLayer: 0,
					LayerCount:     1,
				},
				SrcOffset: common.Offset3D{0, 0, 0},
				DstOffset: common.Offset3D{0, 0, 0},
				Extent:    common.Extent3D{i.Width, i.Height, 1},
			},
		})
	if err != nil {
		return err
	}

	err = i.SetImageLayout(mappableImage, core1_0.AspectColor, core1_0.ImageLayoutTransferDstOptimal, core1_0.ImageLayoutGeneral, core1_0.PipelineStageTransfer, core1_0.PipelineStageHost)
	if err != nil {
		return err
	}

	_, err = i.Cmd.End()
	if err != nil {
		return err
	}

	cmdFence, _, err := i.Loader.CreateFence(i.Device, nil, core1_0.FenceOptions{})
	if err != nil {
		return err
	}

	_, err = i.GraphicsQueue.SubmitToQueue(cmdFence, []core1_0.SubmitOptions{
		{
			CommandBuffers: []core1_0.CommandBuffer{i.Cmd},
		},
	})
	if err != nil {
		return err
	}

	for {
		res, err := i.Device.WaitForFences(true, FenceTimeout, []core1_0.Fence{cmdFence})
		if err != nil {
			return err
		}

		if res != core1_0.VKTimeout {
			break
		}
	}

	cmdFence.Destroy(nil)

	filename := fmt.Sprintf("%s.png", baseName)
	subresourceLayout := mappableImage.SubresourceLayout(&common.ImageSubresource{
		AspectMask: core1_0.AspectColor,
		MipLevel:   0,
		ArrayLayer: 0,
	})

	memPtr, _, err := mappableMemory.MapMemory(0, memReqs.Size, 0)
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
		if i.Format == core1_0.DataFormatB8G8R8A8UnsignedNormalized || i.Format == core1_0.DataFormatB8G8R8A8SRGB {
			for x := 0; x < i.Width; x++ {
				outImg.Set(x, y, color.RGBA{
					B: dataBuffer[rowIndex],
					G: dataBuffer[rowIndex+1],
					R: dataBuffer[rowIndex+2],
					A: dataBuffer[rowIndex+3],
				})
				rowIndex += 4
			}
		} else if i.Format == core1_0.DataFormatR8G8B8A8UnsignedNormalized {
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

	mappableMemory.UnmapMemory()
	mappableImage.Destroy(nil)
	mappableMemory.Free(nil)

	writeFile, err := os.Create(filename)
	if err != nil {
		return err
	}

	return png.Encode(writeFile, outImg)
}

func (i *SampleInfo) InitTextureBuffer(textureObj *TextureObject) error {
	var err error
	textureObj.Buffer, _, err = i.Loader.CreateBuffer(i.Device, nil, core1_0.BufferOptions{
		BufferSize:  textureObj.TexWidth * textureObj.TexHeight * 4,
		Usage:       core1_0.BufferUsageTransferSrc,
		SharingMode: core1_0.SharingExclusive,
	})
	if err != nil {
		return err
	}

	memReqs := textureObj.Buffer.MemoryRequirements()
	textureObj.BufferSize = memReqs.Size

	requirements := core1_0.MemoryPropertyHostVisible | core1_0.MemoryPropertyHostCoherent
	memoryIndex, err := i.MemoryTypeFromProperties(memReqs.MemoryType, requirements)
	if err != nil {
		return err
	}

	/* allocate memory */
	textureObj.BufferMemory, _, err = i.Loader.AllocateMemory(i.Device, nil, core1_0.DeviceMemoryOptions{
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memoryIndex,
	})
	if err != nil {
		return err
	}

	_, err = textureObj.Buffer.BindBufferMemory(textureObj.BufferMemory, 0)
	return err
}

func (i *SampleInfo) InitImage(textureReader io.Reader, extraUsages common.ImageUsages, extraFeatures common.FormatFeatures) (*TextureObject, error) {
	image, _, err := image.Decode(textureReader)
	if err != nil {
		return nil, err
	}

	textureObj := &TextureObject{}
	textureObj.TexWidth = image.Bounds().Size().X
	textureObj.TexHeight = image.Bounds().Size().Y

	formatProps := i.Gpus[0].FormatProperties(core1_0.DataFormatR8G8B8A8UnsignedNormalized)

	/* See if we can use a linear tiled image for a texture, if not, we will
	 * need a staging buffer for the texture data */
	allFeatures := core1_0.FormatFeatureSampledImage | extraFeatures
	textureObj.NeedsStaging = (formatProps.LinearTilingFeatures & allFeatures) != allFeatures

	if textureObj.NeedsStaging {
		if (formatProps.OptimalTilingFeatures & allFeatures) != allFeatures {
			return nil, errors.Newf("Format %s cannot support featureset %s\n", core1_0.DataFormatR8G8B8A8UnsignedNormalized, allFeatures)
		}
		err = i.InitTextureBuffer(textureObj)
		if err != nil {
			return nil, err
		}
		extraUsages |= core1_0.ImageUsageTransferDst
	}

	imageOptions := core1_0.ImageOptions{
		ImageType:   core1_0.ImageType2D,
		Format:      core1_0.DataFormatR8G8B8A8UnsignedNormalized,
		Extent:      common.Extent3D{Width: textureObj.TexWidth, Height: textureObj.TexHeight, Depth: 1},
		MipLevels:   1,
		ArrayLayers: 1,
		Samples:     NumSamples,
		Usage:       core1_0.ImageUsageSampled | extraUsages,
		SharingMode: core1_0.SharingExclusive,
	}
	if textureObj.NeedsStaging {
		imageOptions.Tiling = core1_0.ImageTilingOptimal
		imageOptions.InitialLayout = core1_0.ImageLayoutUndefined
	} else {
		imageOptions.Tiling = core1_0.ImageTilingLinear
		imageOptions.InitialLayout = core1_0.ImageLayoutPreInitialized
	}

	textureObj.Image, _, err = i.Loader.CreateImage(i.Device, nil, imageOptions)
	if err != nil {
		return nil, err
	}

	memReqs := textureObj.Image.MemoryRequirements()

	var requirements common.MemoryProperties
	if !textureObj.NeedsStaging {
		requirements = core1_0.MemoryPropertyHostVisible | core1_0.MemoryPropertyHostCoherent
	}

	memoryIndex, err := i.MemoryTypeFromProperties(memReqs.MemoryType, requirements)
	if err != nil {
		return nil, err
	}

	/* allocate memory */
	textureObj.ImageMemory, _, err = i.Loader.AllocateMemory(i.Device, nil, core1_0.DeviceMemoryOptions{
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memoryIndex,
	})
	if err != nil {
		return nil, err
	}

	/* bind memory */
	_, err = textureObj.Image.BindImageMemory(textureObj.ImageMemory, 0)
	if err != nil {
		return nil, err
	}

	_, err = i.Cmd.End()
	if err != nil {
		return nil, err
	}

	cmdFence, _, err := i.Loader.CreateFence(i.Device, nil, core1_0.FenceOptions{})
	if err != nil {
		return nil, err
	}

	/* Queue the command buffer for execution */
	_, err = i.GraphicsQueue.SubmitToQueue(cmdFence, []core1_0.SubmitOptions{
		{
			CommandBuffers: []core1_0.CommandBuffer{i.Cmd},
		},
	})
	if err != nil {
		return nil, err
	}

	subResource := &common.ImageSubresource{
		AspectMask: core1_0.AspectColor,
		MipLevel:   0,
		ArrayLayer: 0,
	}
	layout := &common.SubresourceLayout{}
	if !textureObj.NeedsStaging {
		/* Get the subresource layout so we know what the row pitch is */
		layout = textureObj.Image.SubresourceLayout(subResource)
	}

	/* Make sure command buffer is finished before mapping */
	for {
		res, err := cmdFence.Wait(FenceTimeout)
		if err != nil {
			return nil, err
		}

		if res != core1_0.VKTimeout {
			break
		}
	}

	cmdFence.Destroy(nil)

	var dataPtr unsafe.Pointer
	var data []byte
	if textureObj.NeedsStaging {
		dataPtr, _, err = textureObj.BufferMemory.MapMemory(0, textureObj.BufferSize, 0)
		data = ([]byte)(unsafe.Slice((*byte)(dataPtr), textureObj.BufferSize))
	} else {
		dataPtr, _, err = textureObj.ImageMemory.MapMemory(0, memReqs.Size, 0)
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
		textureObj.BufferMemory.UnmapMemory()
	} else {
		textureObj.ImageMemory.UnmapMemory()
	}

	_, err = i.Cmd.Reset(0)
	if err != nil {
		return nil, err
	}
	_, err = i.Cmd.Begin(core1_0.BeginOptions{})
	if err != nil {
		return nil, err
	}

	if !textureObj.NeedsStaging {
		/* If we can use the linear tiled image as a texture, just do it */
		textureObj.ImageLayout = core1_0.ImageLayoutShaderReadOnlyOptimal
		err = i.SetImageLayout(textureObj.Image, core1_0.AspectColor, core1_0.ImageLayoutPreInitialized, textureObj.ImageLayout, core1_0.PipelineStageHost, core1_0.PipelineStageFragmentShader)
		if err != nil {
			return nil, err
		}
	} else {
		/* Since we're going to blit to the texture image, set its layout to
		 * DESTINATION_OPTIMAL */
		err = i.SetImageLayout(textureObj.Image, core1_0.AspectColor, core1_0.ImageLayoutUndefined, core1_0.ImageLayoutTransferDstOptimal, core1_0.PipelineStageTopOfPipe, core1_0.PipelineStageTransfer)
		if err != nil {
			return nil, err
		}

		/* Put the copy command into the command buffer */
		err = i.Cmd.CmdCopyBufferToImage(textureObj.Buffer, textureObj.Image, core1_0.ImageLayoutTransferDstOptimal, []core1_0.BufferImageCopy{
			{
				BufferOffset:      0,
				BufferRowLength:   textureObj.TexWidth,
				BufferImageHeight: textureObj.TexHeight,
				ImageSubresource: common.ImageSubresourceLayers{
					AspectMask:     core1_0.AspectColor,
					MipLevel:       0,
					BaseArrayLayer: 0,
					LayerCount:     1,
				},
				ImageOffset: common.Offset3D{0, 0, 0},
				ImageExtent: common.Extent3D{textureObj.TexWidth, textureObj.TexHeight, 1},
			},
		})
		if err != nil {
			return nil, err
		}

		/* Set the layout for the texture image from DESTINATION_OPTIMAL to
		 * SHADER_READ_ONLY */
		textureObj.ImageLayout = core1_0.ImageLayoutShaderReadOnlyOptimal
		err = i.SetImageLayout(textureObj.Image, core1_0.AspectColor, core1_0.ImageLayoutTransferDstOptimal, textureObj.ImageLayout, core1_0.PipelineStageTransfer, core1_0.PipelineStageFragmentShader)
		if err != nil {
			return nil, err
		}
	}

	/* create image view */
	textureObj.View, _, err = i.Loader.CreateImageView(i.Device, nil, core1_0.ImageViewOptions{
		Image:    textureObj.Image,
		ViewType: core1_0.ViewType2D,
		Format:   core1_0.DataFormatR8G8B8A8UnsignedNormalized,
		Components: core1_0.ComponentMapping{
			R: core1_0.SwizzleRed,
			G: core1_0.SwizzleGreen,
			B: core1_0.SwizzleBlue,
			A: core1_0.SwizzleAlpha,
		},
		SubresourceRange: common.ImageSubresourceRange{
			AspectMask:     core1_0.AspectColor,
			BaseMipLevel:   0,
			LevelCount:     1,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
	})
	return textureObj, err
}

func (i *SampleInfo) InitTexture(textureReader io.Reader, extraUsages common.ImageUsages, extraFeatures common.FormatFeatures) error {
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
		i.Textures[ind].Sampler.Destroy(nil)
		i.Textures[ind].View.Destroy(nil)
		i.Textures[ind].Image.Destroy(nil)
		i.Textures[ind].ImageMemory.Free(nil)

		if i.Textures[ind].Buffer != nil {
			i.Textures[ind].Buffer.Destroy(nil)
		}

		if i.Textures[ind].BufferMemory != nil {
			i.Textures[ind].BufferMemory.Free(nil)
		}
	}
}
