package utils

import (
	"fmt"
	"github.com/CannibalVox/VKng/core"
	"github.com/CannibalVox/VKng/core/common"
	"github.com/cockroachdb/errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"unsafe"
)

func (i *SampleInfo) setImageLayout(image core.Image, aspectMask common.ImageAspectFlags, oldImageLayout common.ImageLayout, newImageLayout common.ImageLayout, sourceStages common.PipelineStages, destStages common.PipelineStages) error {
	imageBarrierOptions := &core.ImageMemoryBarrierOptions{
		OldLayout:            oldImageLayout,
		NewLayout:            newImageLayout,
		SrcQueueFamilyIndex:  -1,
		DestQueueFamilyIndex: -1,
		Image:                image,
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
		imageBarrierOptions.DestAccessMask = common.AccessTransferWrite
	case common.LayoutTransferSrcOptimal:
		imageBarrierOptions.DestAccessMask = common.AccessTransferRead
	case common.LayoutShaderReadOnlyOptimal:
		imageBarrierOptions.DestAccessMask = common.AccessShaderRead
	case common.LayoutColorAttachmentOptimal:
		imageBarrierOptions.DestAccessMask = common.AccessColorAttachmentWrite
	case common.LayoutDepthStencilAttachmentOptimal:
		imageBarrierOptions.DestAccessMask = common.AccessDepthStencilAttachmentWrite
	}

	return i.Cmd.CmdPipelineBarrier(sourceStages, destStages, 0, nil, nil, []*core.ImageMemoryBarrierOptions{imageBarrierOptions})
}

func (i *SampleInfo) WritePNG(baseName string) error {
	mappableImage, _, err := i.Loader.CreateImage(i.Device, &core.ImageOptions{
		Type:   common.ImageType2D,
		Format: i.Format,
		Extent: common.Extent3D{
			Width:  i.Width,
			Height: i.Height,
			Depth:  1,
		},
		MipLevels:     1,
		ArrayLayers:   1,
		Samples:       common.Samples1,
		Tiling:        common.ImageTilingLinear,
		Usage:         common.ImageTransferDest,
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

	err = i.setImageLayout(mappableImage, common.AspectColor, common.LayoutUndefined, common.LayoutTransferDstOptimal, common.PipelineStageTopOfPipe, common.PipelineStageTransfer)
	if err != nil {
		return err
	}

	err = i.setImageLayout(i.Buffers[i.CurrentBuffer].Image, common.AspectColor, common.LayoutPresentSrcKHR, common.LayoutTransferSrcOptimal, common.PipelineStageBottomOfPipe, common.PipelineStageTransfer)
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

	err = i.setImageLayout(mappableImage, common.AspectColor, common.LayoutTransferDstOptimal, common.LayoutGeneral, common.PipelineStageTransfer, common.PipelineStageHost)
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
