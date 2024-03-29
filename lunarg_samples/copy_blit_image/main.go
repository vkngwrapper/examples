package main

import (
	"github.com/loov/hrtime"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/vkngwrapper/core/v2"
	"github.com/vkngwrapper/core/v2/common"
	"github.com/vkngwrapper/core/v2/core1_0"
	"github.com/vkngwrapper/examples/lunarg_samples/utils"
	"github.com/vkngwrapper/extensions/v2/ext_debug_utils"
	"github.com/vkngwrapper/extensions/v2/khr_swapchain"
	"log"
	"runtime"
	"runtime/debug"
	"time"
	"unsafe"
)

func logDebug(msgType ext_debug_utils.DebugUtilsMessageTypeFlags, severity ext_debug_utils.DebugUtilsMessageSeverityFlags, data *ext_debug_utils.DebugUtilsMessengerCallbackData) bool {
	log.Printf("[%s %s] - %s", severity, msgType, data.Message)

	if (severity & ext_debug_utils.SeverityError) != 0 {
		debug.PrintStack()
	}

	log.Println()
	return false
}

/*
VULKAN_SAMPLE_SHORT_DESCRIPTION
Copy/blit image
*/

/* Create a checkerboard image, and blit a small area of it to the
 * presentation image. We should see bigger sqaures.  Then copy part of
 * the checkboard to the presentation image - we should see small squares
 */

func main() {
	runtime.LockOSThread()

	info := &utils.SampleInfo{}
	err := info.ProcessCommandLineArgs()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitWindowSize(640, 640)
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitWindow()
	if err != nil {
		log.Fatalln(err)
	}

	info.Loader, err = core.CreateLoaderFromProcAddr(sdl.VulkanGetVkGetInstanceProcAddr())
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitGlobalLayerProperties()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitInstanceExtensionNames()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitDeviceExtensionNames()
	if err != nil {
		log.Fatalln(err)
	}

	info.InstanceExtensionNames = append(info.InstanceExtensionNames, ext_debug_utils.ExtensionName)
	info.InstanceLayerNames = append(info.InstanceLayerNames, "VK_LAYER_KHRONOS_validation")
	debugOptions := ext_debug_utils.DebugUtilsMessengerCreateInfo{
		MessageSeverity: ext_debug_utils.SeverityWarning | ext_debug_utils.SeverityError,
		MessageType:     ext_debug_utils.TypeGeneral | ext_debug_utils.TypeValidation | ext_debug_utils.TypePerformance,
		UserCallback:    logDebug,
	}

	err = info.InitInstance("Copy/Blit Image", debugOptions)
	if err != nil {
		log.Fatalln(err)
	}

	debugLoader := ext_debug_utils.CreateExtensionFromInstance(info.Instance)
	debugMessenger, _, err := debugLoader.CreateDebugUtilsMessenger(info.Instance, nil, debugOptions)
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitEnumerateDevice()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitSwapchainExtension()
	if err != nil {
		log.Fatalln(err)
	}

	surfCapabilities, _, err := info.Surface.PhysicalDeviceSurfaceCapabilities(info.Gpus[0])
	if err != nil {
		log.Fatalln(err)
	}

	if (surfCapabilities.SupportedUsageFlags & core1_0.ImageUsageTransferDst) == 0 {
		log.Fatalln("Surface cannot be destination of blit - abort")
	}

	err = info.InitDevice()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitCommandPool()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitCommandBuffer()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.ExecuteBeginCommandBuffer()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitDeviceQueue()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitSwapchain(core1_0.ImageUsageColorAttachment | core1_0.ImageUsageTransferDst)
	if err != nil {
		log.Fatalln(err)
	}

	/* VULKAN_KEY_START */
	formatProps := info.Gpus[0].FormatProperties(info.Format)
	if (formatProps.LinearTilingFeatures & core1_0.FormatFeatureBlitSource) == 0 {
		log.Fatalln("Format cannot be used as transfer source")
	}

	imageAcquiredSemaphore, _, err := info.Device.CreateSemaphore(nil, core1_0.SemaphoreCreateInfo{})
	if err != nil {
		log.Fatalln(err)
	}

	// Get the index of the next available swapchain image:
	info.CurrentBuffer, _, err = info.Swapchain.AcquireNextImage(common.NoTimeout, imageAcquiredSemaphore, nil)
	// TODO: Deal with the VK_SUBOPTIMAL_KHR and VK_ERROR_OUT_OF_DATE_KHR
	// return codes
	if err != nil {
		log.Fatalln(err)
	}

	// We'll be blitting into the presentable image, set the layout accordingly
	err = info.SetImageLayout(info.Buffers[info.CurrentBuffer].Image, core1_0.ImageAspectColor, core1_0.ImageLayoutUndefined, core1_0.ImageLayoutTransferDstOptimal, core1_0.PipelineStageTopOfPipe, core1_0.PipelineStageTransfer)
	if err != nil {
		log.Fatalln(err)
	}

	// Create an image, map it, and write some values to the image
	bltSrcImage, _, err := info.Device.CreateImage(nil, core1_0.ImageCreateInfo{
		ImageType:     core1_0.ImageType2D,
		Format:        info.Format,
		Extent:        core1_0.Extent3D{Width: info.Width, Height: info.Height, Depth: 1},
		MipLevels:     1,
		ArrayLayers:   1,
		Samples:       utils.NumSamples,
		SharingMode:   core1_0.SharingModeExclusive,
		Usage:         core1_0.ImageUsageTransferSrc,
		Tiling:        core1_0.ImageTilingLinear,
		InitialLayout: core1_0.ImageLayoutUndefined,
	})
	if err != nil {
		log.Fatalln(err)
	}

	memReq := bltSrcImage.MemoryRequirements()
	memoryIndex, err := info.MemoryTypeFromProperties(memReq.MemoryTypeBits, core1_0.MemoryPropertyHostVisible)
	if err != nil {
		log.Fatalln(err)
	}

	dmem, _, err := info.Device.AllocateMemory(nil, core1_0.MemoryAllocateInfo{
		AllocationSize:  memReq.Size,
		MemoryTypeIndex: memoryIndex,
	})
	if err != nil {
		log.Fatalln(err)
	}
	_, err = bltSrcImage.BindImageMemory(dmem, 0)
	if err != nil {
		log.Fatalln(err)
	}

	err = info.SetImageLayout(bltSrcImage, core1_0.ImageAspectColor, core1_0.ImageLayoutUndefined, core1_0.ImageLayoutGeneral, core1_0.PipelineStageTopOfPipe, core1_0.PipelineStageHost)
	if err != nil {
		log.Fatalln(err)
	}

	_, err = info.Cmd.End()
	if err != nil {
		log.Fatalln(err)
	}

	cmdFence, err := info.InitFence()
	if err != nil {
		log.Fatalln(err)
	}

	/* Queue the command buffer for execution */
	_, err = info.GraphicsQueue.Submit(cmdFence, []core1_0.SubmitInfo{
		{
			WaitDstStageMask: []core1_0.PipelineStageFlags{core1_0.PipelineStageColorAttachmentOutput},
			WaitSemaphores:   []core1_0.Semaphore{imageAcquiredSemaphore},
			CommandBuffers:   []core1_0.CommandBuffer{info.Cmd},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	/* Make sure command buffer is finished before mapping */
	for {
		res, err := cmdFence.Wait(common.NoTimeout)
		if err != nil {
			log.Fatalln(err)
		}

		if res != core1_0.VKTimeout {
			break
		}
	}
	cmdFence.Destroy(nil)

	pImgMem, _, err := dmem.Map(0, memReq.Size, 0)
	if err != nil {
		log.Fatalln(err)
	}

	imgBytes := ([]byte)(unsafe.Slice((*byte)(pImgMem), info.Height*info.Width*4))
	imgByteIndex := 0

	for row := 0; row < info.Height; row++ {
		for col := 0; col < info.Width; col++ {
			setVal := ((row & 0x8) ^ (col & 0x8)) >> 3
			rgb := byte(setVal * 255)
			imgBytes[imgByteIndex] = rgb
			imgBytes[imgByteIndex+1] = rgb
			imgBytes[imgByteIndex+2] = rgb
			imgBytes[imgByteIndex+3] = 255
			imgByteIndex += 4
		}
	}

	// Flush the mapped memory and then unmap it  Assume it isn't coherent since
	// we didn't really confirm
	_, err = info.Device.FlushMappedMemoryRanges([]core1_0.MappedMemoryRange{
		{
			Memory: dmem,
			Offset: 0,
			Size:   memReq.Size,
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	dmem.Unmap()

	_, err = info.Cmd.Reset(0)
	if err != nil {
		log.Fatalln(err)
	}
	err = info.ExecuteBeginCommandBuffer()
	if err != nil {
		log.Fatalln(err)
	}
	// Intend to blit from this image, set the layout accordingly
	err = info.SetImageLayout(bltSrcImage, core1_0.ImageAspectColor, core1_0.ImageLayoutGeneral, core1_0.ImageLayoutTransferSrcOptimal, core1_0.PipelineStageHost, core1_0.PipelineStageTransfer)
	if err != nil {
		log.Fatalln(err)
	}

	bltDstImage := info.Buffers[info.CurrentBuffer].Image

	// Do a 32x32 blit to all of the dst image - should get big squares
	err = info.Cmd.CmdBlitImage(bltSrcImage, core1_0.ImageLayoutTransferSrcOptimal, bltDstImage, core1_0.ImageLayoutTransferDstOptimal, []core1_0.ImageBlit{
		{
			SrcSubresource: core1_0.ImageSubresourceLayers{
				AspectMask:     core1_0.ImageAspectColor,
				MipLevel:       0,
				BaseArrayLayer: 0,
				LayerCount:     1,
			},
			SrcOffsets: [2]core1_0.Offset3D{
				{X: 0, Y: 0, Z: 0},
				{X: 32, Y: 32, Z: 1},
			},
			DstSubresource: core1_0.ImageSubresourceLayers{
				AspectMask:     core1_0.ImageAspectColor,
				MipLevel:       0,
				BaseArrayLayer: 0,
				LayerCount:     1,
			},
			DstOffsets: [2]core1_0.Offset3D{
				{X: 0, Y: 0, Z: 0},
				{X: info.Width, Y: info.Height, Z: 1},
			},
		},
	}, core1_0.FilterLinear)

	if err != nil {
		log.Fatalln(err)
	}

	// Use a barrier to make sure the blit is finished before the copy starts
	err = info.Cmd.CmdPipelineBarrier(core1_0.PipelineStageTransfer, core1_0.PipelineStageTransfer, 0, nil, nil, []core1_0.ImageMemoryBarrier{
		{
			SrcAccessMask:       core1_0.AccessTransferWrite,
			DstAccessMask:       core1_0.AccessMemoryRead,
			OldLayout:           core1_0.ImageLayoutTransferDstOptimal,
			NewLayout:           core1_0.ImageLayoutTransferDstOptimal,
			SrcQueueFamilyIndex: -1,
			DstQueueFamilyIndex: -1,
			SubresourceRange: core1_0.ImageSubresourceRange{
				AspectMask:     core1_0.ImageAspectColor,
				BaseMipLevel:   0,
				LevelCount:     1,
				BaseArrayLayer: 0,
				LayerCount:     1,
			},
			Image: bltDstImage,
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Do a image copy to part of the dst image - checks should stay small
	err = info.Cmd.CmdCopyImage(bltSrcImage, core1_0.ImageLayoutTransferSrcOptimal, bltDstImage, core1_0.ImageLayoutTransferDstOptimal, []core1_0.ImageCopy{
		{
			SrcSubresource: core1_0.ImageSubresourceLayers{
				AspectMask:     core1_0.ImageAspectColor,
				MipLevel:       0,
				BaseArrayLayer: 0,
				LayerCount:     1,
			},
			SrcOffset: core1_0.Offset3D{X: 0, Y: 0, Z: 0},
			DstSubresource: core1_0.ImageSubresourceLayers{
				AspectMask:     core1_0.ImageAspectColor,
				MipLevel:       0,
				BaseArrayLayer: 0,
				LayerCount:     1,
			},
			DstOffset: core1_0.Offset3D{X: 256, Y: 256, Z: 0},
			Extent:    core1_0.Extent3D{Width: 128, Height: 128, Depth: 1},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	err = info.Cmd.CmdPipelineBarrier(core1_0.PipelineStageTransfer, core1_0.PipelineStageBottomOfPipe, 0, nil, nil, []core1_0.ImageMemoryBarrier{
		{
			SrcAccessMask:       core1_0.AccessTransferWrite,
			DstAccessMask:       core1_0.AccessMemoryRead,
			OldLayout:           core1_0.ImageLayoutTransferDstOptimal,
			NewLayout:           khr_swapchain.ImageLayoutPresentSrc,
			SrcQueueFamilyIndex: -1,
			DstQueueFamilyIndex: -1,
			SubresourceRange: core1_0.ImageSubresourceRange{
				AspectMask:     core1_0.ImageAspectColor,
				BaseMipLevel:   0,
				LevelCount:     1,
				BaseArrayLayer: 0,
				LayerCount:     1,
			},
			Image: info.Buffers[info.CurrentBuffer].Image,
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	_, err = info.Cmd.End()
	if err != nil {
		log.Fatalln(err)
	}

	drawFence, _, err := info.Device.CreateFence(nil, core1_0.FenceCreateInfo{})
	if err != nil {
		log.Fatalln(err)
	}

	/* Queue the command buffer for execution */
	_, err = info.GraphicsQueue.Submit(drawFence, []core1_0.SubmitInfo{
		{
			CommandBuffers: []core1_0.CommandBuffer{info.Cmd},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	_, err = info.GraphicsQueue.WaitIdle()
	if err != nil {
		log.Fatalln(err)
	}

	/* Now present the image in the window */

	/* Make sure command buffer is finished before presenting */
	for {
		res, err := drawFence.Wait(utils.FenceTimeout)
		if err != nil {
			log.Fatalln(err)
		}

		if res != core1_0.VKTimeout {
			break
		}
	}
	_, err = info.SwapchainExtension.QueuePresent(info.PresentQueue, khr_swapchain.PresentInfo{
		Swapchains:   []khr_swapchain.Swapchain{info.Swapchain},
		ImageIndices: []int{info.CurrentBuffer},
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Some platforms, like mac, will not finish the present until an event poll
	start := hrtime.Now()
	for hrtime.Since(start) < 5*time.Second {
		sdl.PollEvent()
	}

	/* VULKAN_KEY_END */
	if info.SaveImages {
		err = info.WritePNG("copy_blit_image")
		if err != nil {
			log.Fatalln(err)
		}
	}

	imageAcquiredSemaphore.Destroy(nil)
	drawFence.Destroy(nil)
	bltSrcImage.Destroy(nil)
	dmem.Free(nil)
	info.DestroySwapchain()
	info.DestroyCommandBuffer()
	info.DestroyCommandPool()
	err = info.DestroyDevice()
	if err != nil {
		log.Fatalln(err)
	}
	info.Surface.Destroy(nil)
	debugMessenger.Destroy(nil)
	info.DestroyInstance()
	err = info.Window.Destroy()
	if err != nil {
		log.Fatalln(err)
	}
}
