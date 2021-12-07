package main

import (
	"embed"
	"github.com/CannibalVox/VKng/core"
	"github.com/CannibalVox/VKng/core/common"
	"github.com/CannibalVox/VKng/examples/lunarg_samples/utils"
	"github.com/CannibalVox/VKng/extensions/ext_debug_utils"
	"github.com/veandco/go-sdl2/sdl"
	"log"
	"runtime/debug"
	"time"
)

//go:embed shaders
var fileSystem embed.FS

func logDebug(msgType ext_debug_utils.MessageType, severity ext_debug_utils.MessageSeverity, data *ext_debug_utils.CallbackData) bool {
	log.Printf("[%s %s] - %s", severity, msgType, data.Message)
	debug.PrintStack()
	log.Println()
	return false
}

func main() {
	info := &utils.SampleInfo{}
	err := info.ProcessCommandLineArgs()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitWindowSize(500, 500)
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
	debugOptions := &ext_debug_utils.CreationOptions{
		CaptureSeverities: ext_debug_utils.SeverityWarning | ext_debug_utils.SeverityError,
		CaptureTypes:      ext_debug_utils.TypeAll,
		Callback:          logDebug,
	}

	err = info.InitInstance("Input Attachment Sample", debugOptions)
	if err != nil {
		log.Fatalln(err)
	}

	debugLoader := ext_debug_utils.CreateLoaderFromInstance(info.Instance)
	debugMessenger, _, err := debugLoader.CreateMessenger(info.Instance, debugOptions)
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitEnumerateDevice()
	if err != nil {
		log.Fatalln(err)
	}

	props := info.Gpus[0].FormatProperties(common.FormatR8G8B8A8UnsignedNormalized)
	if (props.OptimalTilingFeatures & common.FormatFeatureColorAttachment) == 0 {
		log.Fatalf("%s format unsupported for input attachment\n", common.FormatR8G8B8A8UnsignedNormalized)
	}

	err = info.InitSwapchainExtension()
	if err != nil {
		log.Fatalln(err)
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

	err = info.InitSwapchain(common.ImageUsageColorAttachment | common.ImageUsageTransferSrc)
	if err != nil {
		log.Fatalln(err)
	}

	/* VULKAN_KEY_START */

	// Create a framebuffer with 2 attachments, one the color attachment
	// the shaders render into, and the other an input attachment which
	// will be cleared to yellow, and then used by the shaders to color
	// the drawn triangle. Final result should be a yellow triangle

	// Create the image that will be used as the input attachment
	// The image for the color attachment is the presentable image already
	// created in init_swapchain()
	inputImage, _, err := info.Loader.CreateImage(info.Device, &core.ImageOptions{
		ImageType:     common.ImageType2D,
		Format:        info.Format,
		Extent:        common.Extent3D{Width: info.Width, Height: info.Height, Depth: 1},
		MipLevels:     1,
		ArrayLayers:   1,
		Samples:       utils.NumSamples,
		Tiling:        common.ImageTilingOptimal,
		InitialLayout: common.LayoutUndefined,
		Usage:         common.ImageUsageInputAttachment | common.ImageUsageTransferDst,
		SharingMode:   common.SharingExclusive,
	})
	if err != nil {
		log.Fatalln(err)
	}

	memReqs := inputImage.MemoryRequirements()
	memoryTypeIndex, err := info.MemoryTypeFromProperties(memReqs.MemoryType, 0)
	if err != nil {
		log.Fatalln(err)
	}

	inputMemory, _, err := info.Device.AllocateMemory(&core.DeviceMemoryOptions{
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memoryTypeIndex,
	})
	if err != nil {
		log.Fatalln(err)
	}

	_, err = inputImage.BindImageMemory(inputMemory, 0)
	if err != nil {
		log.Fatalln(err)
	}

	// Set the image layout to TRANSFER_DST_OPTIMAL to be ready for clear
	err = info.SetImageLayout(inputImage, common.AspectColor, common.LayoutUndefined, common.LayoutTransferDstOptimal, common.PipelineStageTopOfPipe, common.PipelineStageTransfer)
	if err != nil {
		log.Fatalln(err)
	}

	// Clear the input attachment image to yellow
	info.Cmd.CmdClearColorImage(inputImage, common.LayoutTransferDstOptimal, &core.ClearValueFloat{1, 1, 0, 0}, []*common.ImageSubresourceRange{
		{
			AspectMask:     common.AspectColor,
			BaseMipLevel:   0,
			LevelCount:     -1,
			BaseArrayLayer: 0,
			LayerCount:     -1,
		},
	})

	// Set the image layout to SHADER_READONLY_OPTIMAL for use by the shaders
	err = info.SetImageLayout(inputImage, common.AspectColor, common.LayoutTransferDstOptimal, common.LayoutShaderReadOnlyOptimal, common.PipelineStageTransfer, common.PipelineStageFragmentShader)
	if err != nil {
		log.Fatalln(err)
	}

	inputAttachmentView, _, err := info.Loader.CreateImageView(info.Device, &core.ImageViewOptions{
		Image:    inputImage,
		ViewType: common.ViewType2D,
		Format:   info.Format,
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
	if err != nil {
		log.Fatalln(err)
	}

	descLayout, _, err := info.Loader.CreateDescriptorSetLayout(info.Device, &core.DescriptorSetLayoutOptions{
		Bindings: []*core.DescriptorLayoutBinding{
			{
				Binding:         0,
				DescriptorType:  common.DescriptorInputAttachment,
				DescriptorCount: 1,
				StageFlags:      common.StageFragment,
			},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}
	info.DescLayout = []core.DescriptorSetLayout{descLayout}

	info.PipelineLayout, _, err = info.Loader.CreatePipelineLayout(info.Device, &core.PipelineLayoutOptions{
		SetLayouts: info.DescLayout,
	})
	if err != nil {
		log.Fatalln(err)
	}

	attachments := []core.AttachmentDescription{
		// First attachment is the color attachment - clear at the beginning of the
		// renderpass and transition layout to PRESENT_SRC_KHR at the end of
		// renderpass
		{
			Format:         info.Format,
			Samples:        common.Samples1,
			LoadOp:         common.LoadOpClear,
			StoreOp:        common.StoreOpStore,
			StencilLoadOp:  common.LoadOpDontCare,
			StencilStoreOp: common.StoreOpDontCare,
			InitialLayout:  common.LayoutUndefined,
			FinalLayout:    common.LayoutPresentSrcKHR,
		},
		// Second attachment is input attachment.  Once cleared it should have
		// width*height yellow pixels.  Doing a subpassLoad in the fragment shader
		// should give the shader the color at the fragments x,y location
		// from the input attachment
		{
			Format:         info.Format,
			Samples:        common.Samples1,
			LoadOp:         common.LoadOpLoad,
			StoreOp:        common.StoreOpDontCare,
			StencilLoadOp:  common.LoadOpDontCare,
			StencilStoreOp: common.StoreOpDontCare,
			InitialLayout:  common.LayoutShaderReadOnlyOptimal,
			FinalLayout:    common.LayoutShaderReadOnlyOptimal,
		},
	}

	colorRef := common.AttachmentReference{AttachmentIndex: 0, Layout: common.LayoutColorAttachmentOptimal}
	inputRef := common.AttachmentReference{AttachmentIndex: 1, Layout: common.LayoutShaderReadOnlyOptimal}

	subpass := core.SubPass{
		BindPoint:        common.BindGraphics,
		InputAttachments: []common.AttachmentReference{inputRef},
		ColorAttachments: []common.AttachmentReference{colorRef},
	}

	subpassDependency := core.SubPassDependency{
		SrcSubPassIndex: core.SubpassExternal,
		DstSubPassIndex: 0,
		SrcStageMask:    common.PipelineStageColorAttachmentOutput,
		DstStageMask:    common.PipelineStageColorAttachmentOutput,
		SrcAccessMask:   0,
		DstAccessMask:   common.AccessColorAttachmentWrite,
	}

	info.RenderPass, _, err = info.Loader.CreateRenderPass(info.Device, &core.RenderPassOptions{
		Attachments:         attachments,
		SubPasses:           []core.SubPass{subpass},
		SubPassDependencies: []core.SubPassDependency{subpassDependency},
	})
	if err != nil {
		log.Fatalln(err)
	}

	vertShaderBytes, err := fileSystem.ReadFile("shaders/vert.spv")
	if err != nil {
		log.Fatalln(err)
	}

	fragShaderBytes, err := fileSystem.ReadFile("shaders/frag.spv")
	if err != nil {
		log.Fatal(err)
	}

	err = info.InitShaders(vertShaderBytes, fragShaderBytes)
	if err != nil {
		log.Fatalln(err)
	}

	for i := 0; i < info.SwapchainImageCount; i++ {
		framebuffer, _, err := info.Loader.CreateFrameBuffer(info.Device, &core.FramebufferOptions{
			RenderPass:  info.RenderPass,
			Attachments: []core.ImageView{info.Buffers[i].View, inputAttachmentView},
			Width:       info.Width,
			Height:      info.Height,
			Layers:      1,
		})
		if err != nil {
			log.Fatalln(err)
		}
		info.Framebuffer = append(info.Framebuffer, framebuffer)
	}

	info.DescPool, _, err = info.Loader.CreateDescriptorPool(info.Device, &core.DescriptorPoolOptions{
		MaxSets: 1,
		PoolSizes: []core.PoolSize{
			{
				Type:            common.DescriptorInputAttachment,
				DescriptorCount: 1,
			},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	info.DescSet, _, err = info.DescPool.AllocateDescriptorSets(&core.DescriptorSetOptions{
		AllocationLayouts: []core.DescriptorSetLayout{descLayout},
	})
	if err != nil {
		log.Fatalln(err)
	}

	err = info.Device.UpdateDescriptorSets([]core.WriteDescriptorSetOptions{
		{
			DstSet:          info.DescSet[0],
			DstBinding:      0,
			DstArrayElement: 0,
			DescriptorType:  common.DescriptorInputAttachment,
			ImageInfo: []core.DescriptorImageInfo{
				{
					ImageLayout: common.LayoutShaderReadOnlyOptimal,
					ImageView:   inputAttachmentView,
				},
			},
		},
	}, nil)
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitPipelineCache()
	if err != nil {
		log.Fatalln(err)
	}
	err = info.InitPipeline(true, false)
	if err != nil {
		log.Fatalln(err)
	}

	// Color attachment clear to gray
	info.ImageAcquiredSemaphore, _, err = info.Loader.CreateSemaphore(info.Device, &core.SemaphoreOptions{})
	if err != nil {
		log.Fatalln(err)
	}

	// Get the index of the next available swapchain image:
	info.CurrentBuffer, _, err = info.Swapchain.AcquireNextImage(common.NoTimeout, info.ImageAcquiredSemaphore, nil)
	// TODO: Deal with the VK_SUBOPTIMAL_KHR and VK_ERROR_OUT_OF_DATE_KHR
	// return codes
	if err != nil {
		log.Fatalln(err)
	}

	err = info.Cmd.CmdBeginRenderPass(core.ContentsInline, &core.RenderPassBeginOptions{
		RenderPass:  info.RenderPass,
		Framebuffer: info.Framebuffer[info.CurrentBuffer],
		RenderArea: common.Rect2D{
			Offset: common.Offset2D{X: 0, Y: 0},
			Extent: common.Extent2D{Width: info.Width, Height: info.Height},
		},
		ClearValues: []core.ClearValue{
			core.ClearValueFloat{0.2, 0.2, 0.2, 0.2},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}
	info.Cmd.CmdBindPipeline(common.BindGraphics, info.Pipeline)
	info.Cmd.CmdBindDescriptorSets(common.BindGraphics, info.PipelineLayout, 0, info.DescSet, nil)

	info.InitViewports()
	info.InitScissors()

	info.Cmd.CmdDraw(3, 1, 0, 0)

	info.Cmd.CmdEndRenderPass()
	_, err = info.Cmd.End()
	if err != nil {
		log.Fatalln(err)
	}

	/* VULKAN_KEY_END */

	drawFence, _, err := info.Loader.CreateFence(info.Device, &core.FenceOptions{})
	if err != nil {
		log.Fatalln(err)
	}

	err = info.ExecuteQueueCmdBuf([]core.CommandBuffer{info.Cmd}, drawFence)
	if err != nil {
		log.Fatalln(err)
	}

	for {
		res, err := drawFence.Wait(utils.FenceTimeout)
		if err != nil {
			log.Fatalln(err)
		}

		if res != core.VKTimeout {
			break
		}
	}
	drawFence.Destroy()

	err = info.ExecutePresentImage()
	if err != nil {
		log.Fatalln(err)
	}

	time.Sleep(time.Second)

	if info.SaveImages {
		err = info.WritePNG("input_attachment")
		if err != nil {
			log.Fatalln(err)
		}
	}

	info.ImageAcquiredSemaphore.Destroy()
	inputAttachmentView.Destroy()
	inputImage.Destroy()
	info.Device.FreeMemory(inputMemory)
	info.DestroyPipeline()
	info.DestroyPipelineCache()
	info.DestroyDescriptorPool()
	info.DestroyFramebuffers()
	info.DestroyShaders()
	info.DestroyRenderpass()
	info.DestroyDescriptorAndPipelineLayouts()
	info.DestroySwapchain()
	info.DestroyCommandBuffer()
	info.DestroyCommandPool()
	err = info.DestroyDevice()
	if err != nil {
		log.Fatalln(err)
	}
	info.Surface.Destroy()
	debugMessenger.Destroy()
	info.DestroyInstance()
	err = info.Window.Destroy()
	if err != nil {
		log.Fatalln(err)
	}
}
