package utils

import (
	"fmt"
	"github.com/CannibalVox/VKng/core"
	"github.com/CannibalVox/VKng/core/common"
	"github.com/cockroachdb/errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"unsafe"
)

func (i *SampleInfo) SetImageLayout(image core.Image, aspectMask common.ImageAspectFlags, oldImageLayout common.ImageLayout, newImageLayout common.ImageLayout, sourceStages common.PipelineStages, destStages common.PipelineStages) error {
	imageBarrierOptions := &core.ImageMemoryBarrierOptions{
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
	case common.LayoutColorAttachmentOptimal:
		imageBarrierOptions.SrcAccessMask = common.AccessColorAttachmentWrite
	case common.LayoutTransferDstOptimal:
		imageBarrierOptions.SrcAccessMask = common.AccessTransferWrite
	case common.LayoutPreInitialized:
		imageBarrierOptions.SrcAccessMask = common.AccessHostWrite
	}

	switch newImageLayout {
	case common.LayoutTransferDstOptimal:
		imageBarrierOptions.DstAccessMask = common.AccessTransferWrite
	case common.LayoutTransferSrcOptimal:
		imageBarrierOptions.DstAccessMask = common.AccessTransferRead
	case common.LayoutShaderReadOnlyOptimal:
		imageBarrierOptions.DstAccessMask = common.AccessShaderRead
	case common.LayoutColorAttachmentOptimal:
		imageBarrierOptions.DstAccessMask = common.AccessColorAttachmentWrite
	case common.LayoutDepthStencilAttachmentOptimal:
		imageBarrierOptions.DstAccessMask = common.AccessDepthStencilAttachmentWrite
	}

	return i.Cmd.CmdPipelineBarrier(sourceStages, destStages, 0, nil, nil, []*core.ImageMemoryBarrierOptions{imageBarrierOptions})
}

func (i *SampleInfo) WritePNG(baseName string) error {
	mappableImage, _, err := i.Loader.CreateImage(i.Device, &core.ImageOptions{
		ImageType: common.ImageType2D,
		Format:    i.Format,
		Extent: common.Extent3D{
			Width:  i.Width,
			Height: i.Height,
			Depth:  1,
		},
		MipLevels:     1,
		ArrayLayers:   1,
		Samples:       common.Samples1,
		Tiling:        common.ImageTilingLinear,
		Usage:         common.ImageUsageTransferDst,
		SharingMode:   common.SharingExclusive,
		InitialLayout: common.LayoutUndefined,
	})
	if err != nil {
		return err
	}

	memReqs := mappableImage.MemoryRequirements()
	memoryTypeIndex, err := i.MemoryTypeFromProperties(memReqs.MemoryType, core.MemoryHostVisible|core.MemoryHostCoherent)
	if err != nil {
		return err
	}

	mappableMemory, _, err := i.Device.AllocateMemory(&core.DeviceMemoryOptions{
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

	_, err = i.Cmd.Begin(&core.BeginOptions{})
	if err != nil {
		return err
	}

	err = i.SetImageLayout(mappableImage, common.AspectColor, common.LayoutUndefined, common.LayoutTransferDstOptimal, common.PipelineStageTopOfPipe, common.PipelineStageTransfer)
	if err != nil {
		return err
	}

	err = i.SetImageLayout(i.Buffers[i.CurrentBuffer].Image, common.AspectColor, common.LayoutPresentSrcKHR, common.LayoutTransferSrcOptimal, common.PipelineStageBottomOfPipe, common.PipelineStageTransfer)
	if err != nil {
		return err
	}

	err = i.Cmd.CmdCopyImage(i.Buffers[i.CurrentBuffer].Image,
		common.LayoutTransferSrcOptimal,
		mappableImage,
		common.LayoutTransferDstOptimal,
		[]core.ImageCopy{
			{
				SrcSubresource: common.ImageSubresourceLayers{
					AspectMask:     common.AspectColor,
					MipLevel:       0,
					BaseArrayLayer: 0,
					LayerCount:     1,
				},
				DstSubresource: common.ImageSubresourceLayers{
					AspectMask:     common.AspectColor,
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

	err = i.SetImageLayout(mappableImage, common.AspectColor, common.LayoutTransferDstOptimal, common.LayoutGeneral, common.PipelineStageTransfer, common.PipelineStageHost)
	if err != nil {
		return err
	}

	_, err = i.Cmd.End()
	if err != nil {
		return err
	}

	cmdFence, _, err := i.Loader.CreateFence(i.Device, &core.FenceOptions{})
	if err != nil {
		return err
	}

	_, err = i.GraphicsQueue.SubmitToQueue(cmdFence, []*core.SubmitOptions{
		{
			CommandBuffers: []core.CommandBuffer{i.Cmd},
		},
	})
	if err != nil {
		return err
	}

	for {
		res, err := i.Device.WaitForFences(true, FenceTimeout, []core.Fence{cmdFence})
		if err != nil {
			return err
		}

		if res != core.VKTimeout {
			break
		}
	}

	cmdFence.Destroy()

	filename := fmt.Sprintf("%s.png", baseName)
	subresourceLayout := mappableImage.SubresourceLayout(&common.ImageSubresource{
		AspectMask: common.AspectColor,
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
		if i.Format == common.FormatB8G8R8A8UnsignedNormalized || i.Format == common.FormatB8G8R8A8SRGB {
			for x := 0; x < i.Width; x++ {
				outImg.Set(x, y, color.RGBA{
					B: dataBuffer[rowIndex],
					G: dataBuffer[rowIndex+1],
					R: dataBuffer[rowIndex+2],
					A: dataBuffer[rowIndex+3],
				})
				rowIndex += 4
			}
		} else if i.Format == common.FormatR8G8B8A8UnsignedNormalized {
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
	mappableImage.Destroy()
	i.Device.FreeMemory(mappableMemory)

	writeFile, err := os.Create(filename)
	if err != nil {
		return err
	}

	return png.Encode(writeFile, outImg)
}

func (i *SampleInfo) InitTextureBuffer(textureObj *TextureObject) error {
	var err error
	textureObj.Buffer, _, err = i.Loader.CreateBuffer(i.Device, &core.BufferOptions{
		BufferSize:  textureObj.TexWidth * textureObj.TexHeight * 4,
		Usage:       common.UsageTransferSrc,
		SharingMode: common.SharingExclusive,
	})
	if err != nil {
		return err
	}

	memReqs := textureObj.Buffer.MemoryRequirements()
	textureObj.BufferSize = memReqs.Size

	requirements := core.MemoryHostVisible | core.MemoryHostCoherent
	memoryIndex, err := i.MemoryTypeFromProperties(memReqs.MemoryType, requirements)
	if err != nil {
		return err
	}

	/* allocate memory */
	textureObj.BufferMemory, _, err = i.Device.AllocateMemory(&core.DeviceMemoryOptions{
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memoryIndex,
	})
	if err != nil {
		return err
	}

	_, err = textureObj.Buffer.BindBufferMemory(textureObj.BufferMemory, 0)
	return err
}

func (i *SampleInfo) InitImage(textureReader io.Reader) (*TextureObject, error) {
	image, _, err := image.Decode(textureReader)
	if err != nil {
		return nil, err
	}

	textureObj := &TextureObject{}
	textureObj.TexWidth = image.Bounds().Size().X
	textureObj.TexHeight = image.Bounds().Size().Y

	formatProps := i.Gpus[0].FormatProperties(common.FormatR8G8B8A8UnsignedNormalized)

	/* See if we can use a linear tiled image for a texture, if not, we will
	 * need a staging buffer for the texture data */
	allFeatures := common.FormatFeatureSampledImage
	var usages common.ImageUsages
	textureObj.NeedsStaging = (formatProps.LinearTilingFeatures & allFeatures) != allFeatures

	if textureObj.NeedsStaging {
		if (formatProps.OptimalTilingFeatures & allFeatures) != allFeatures {
			return nil, errors.Newf("Format %s cannot support featureset %s\n", common.FormatR8G8B8A8UnsignedNormalized, allFeatures)
		}
		err = i.InitTextureBuffer(textureObj)
		if err != nil {
			return nil, err
		}
		usages |= common.ImageUsageTransferDst
	}

	imageOptions := &core.ImageOptions{
		ImageType:   common.ImageType2D,
		Format:      common.FormatR8G8B8A8UnsignedNormalized,
		Extent:      common.Extent3D{Width: textureObj.TexWidth, Height: textureObj.TexHeight, Depth: 1},
		MipLevels:   1,
		ArrayLayers: 1,
		Samples:     NumSamples,
		Usage:       common.ImageUsageSampled | usages,
		SharingMode: common.SharingExclusive,
	}
	if textureObj.NeedsStaging {
		imageOptions.Tiling = common.ImageTilingOptimal
		imageOptions.InitialLayout = common.LayoutUndefined
	} else {
		imageOptions.Tiling = common.ImageTilingLinear
		imageOptions.InitialLayout = common.LayoutPreInitialized
	}

	textureObj.Image, _, err = i.Loader.CreateImage(i.Device, imageOptions)
	if err != nil {
		return nil, err
	}

	memReqs := textureObj.Image.MemoryRequirements()

	var requirements core.MemoryPropertyFlags
	if !textureObj.NeedsStaging {
		requirements = core.MemoryHostVisible | core.MemoryHostCoherent
	}

	memoryIndex, err := i.MemoryTypeFromProperties(memReqs.MemoryType, requirements)
	if err != nil {
		return nil, err
	}

	/* allocate memory */
	textureObj.ImageMemory, _, err = i.Device.AllocateMemory(&core.DeviceMemoryOptions{
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

	cmdFence, _, err := i.Loader.CreateFence(i.Device, &core.FenceOptions{})
	if err != nil {
		return nil, err
	}

	/* Queue the command buffer for execution */
	_, err = i.GraphicsQueue.SubmitToQueue(cmdFence, []*core.SubmitOptions{
		{
			CommandBuffers: []core.CommandBuffer{i.Cmd},
		},
	})
	if err != nil {
		return nil, err
	}

	subResource := &common.ImageSubresource{
		AspectMask: common.AspectColor,
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

		if res != core.VKTimeout {
			break
		}
	}

	cmdFence.Destroy()

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
	_, err = i.Cmd.Begin(&core.BeginOptions{})
	if err != nil {
		return nil, err
	}

	if !textureObj.NeedsStaging {
		/* If we can use the linear tiled image as a texture, just do it */
		textureObj.ImageLayout = common.LayoutShaderReadOnlyOptimal
		err = i.SetImageLayout(textureObj.Image, common.AspectColor, common.LayoutPreInitialized, textureObj.ImageLayout, common.PipelineStageHost, common.PipelineStageFragmentShader)
		if err != nil {
			return nil, err
		}
	} else {
		/* Since we're going to blit to the texture image, set its layout to
		 * DESTINATION_OPTIMAL */
		err = i.SetImageLayout(textureObj.Image, common.AspectColor, common.LayoutUndefined, common.LayoutTransferDstOptimal, common.PipelineStageTopOfPipe, common.PipelineStageTransfer)
		if err != nil {
			return nil, err
		}

		/* Put the copy command into the command buffer */
		err = i.Cmd.CmdCopyBufferToImage(textureObj.Buffer, textureObj.Image, common.LayoutTransferDstOptimal, []*core.BufferImageCopy{
			{
				BufferOffset:      0,
				BufferRowLength:   textureObj.TexWidth,
				BufferImageHeight: textureObj.TexHeight,
				ImageSubresource: common.ImageSubresourceLayers{
					AspectMask:     common.AspectColor,
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
		textureObj.ImageLayout = common.LayoutShaderReadOnlyOptimal
		err = i.SetImageLayout(textureObj.Image, common.AspectColor, common.LayoutTransferDstOptimal, textureObj.ImageLayout, common.PipelineStageTransfer, common.PipelineStageFragmentShader)
		if err != nil {
			return nil, err
		}
	}

	/* create image view */
	textureObj.View, _, err = i.Loader.CreateImageView(i.Device, &core.ImageViewOptions{
		Image:    textureObj.Image,
		ViewType: common.ViewType2D,
		Format:   common.FormatR8G8B8A8UnsignedNormalized,
		Components: common.ComponentMapping{
			R: common.SwizzleRed,
			G: common.SwizzleGreen,
			B: common.SwizzleBlue,
			A: common.SwizzleAlpha,
		},
		SubresourceRange: common.ImageSubresourceRange{
			AspectMask:     common.AspectColor,
			BaseMipLevel:   0,
			LevelCount:     1,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
	})
	return textureObj, err
}
