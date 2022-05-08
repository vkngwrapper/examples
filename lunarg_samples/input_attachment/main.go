package main

import (
	"embed"
	"github.com/CannibalVox/VKng/core"
	"github.com/CannibalVox/VKng/core/common"
	"github.com/CannibalVox/VKng/core/core1_0"
	"github.com/CannibalVox/VKng/examples/lunarg_samples/utils"
	"github.com/CannibalVox/VKng/extensions/ext_debug_utils"
	"github.com/CannibalVox/VKng/extensions/khr_swapchain"
	"github.com/veandco/go-sdl2/sdl"
	"log"
	"runtime/debug"
	"time"
)

//go:embed shaders
var fileSystem embed.FS

func logDebug(msgType ext_debug_utils.MessageTypes, severity ext_debug_utils.MessageSeverities, data *ext_debug_utils.CallbackDataOptions) bool {
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
	debugOptions := ext_debug_utils.CreateOptions{
		CaptureSeverities: ext_debug_utils.SeverityWarning | ext_debug_utils.SeverityError,
		CaptureTypes:      ext_debug_utils.TypeGeneral | ext_debug_utils.TypeValidation | ext_debug_utils.TypePerformance,
		Callback:          logDebug,
	}

	err = info.InitInstance("Input Attachment Sample", debugOptions)
	if err != nil {
		log.Fatalln(err)
	}

	debugLoader := ext_debug_utils.CreateExtensionFromInstance(info.Instance)
	debugMessenger, _, err := debugLoader.CreateMessenger(info.Instance, nil, debugOptions)
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitEnumerateDevice()
	if err != nil {
		log.Fatalln(err)
	}

	props := info.Gpus[0].FormatProperties(core1_0.DataFormatR8G8B8A8UnsignedNormalized)
	if (props.OptimalTilingFeatures & core1_0.FormatFeatureColorAttachment) == 0 {
		log.Fatalf("%s format unsupported for input attachment\n", core1_0.DataFormatR8G8B8A8UnsignedNormalized)
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

	err = info.InitSwapchain(core1_0.ImageUsageColorAttachment | core1_0.ImageUsageTransferSrc)
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
	inputImage, _, err := info.Loader.CreateImage(info.Device, nil, core1_0.ImageCreateOptions{
		ImageType:     core1_0.ImageType2D,
		Format:        info.Format,
		Extent:        common.Extent3D{Width: info.Width, Height: info.Height, Depth: 1},
		MipLevels:     1,
		ArrayLayers:   1,
		Samples:       utils.NumSamples,
		Tiling:        core1_0.ImageTilingOptimal,
		InitialLayout: core1_0.ImageLayoutUndefined,
		Usage:         core1_0.ImageUsageInputAttachment | core1_0.ImageUsageTransferDst,
		SharingMode:   core1_0.SharingExclusive,
	})
	if err != nil {
		log.Fatalln(err)
	}

	memReqs := inputImage.MemoryRequirements()
	memoryTypeIndex, err := info.MemoryTypeFromProperties(memReqs.MemoryType, 0)
	if err != nil {
		log.Fatalln(err)
	}

	inputMemory, _, err := info.Loader.AllocateMemory(info.Device, nil, core1_0.MemoryAllocateOptions{
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
	err = info.SetImageLayout(inputImage, core1_0.AspectColor, core1_0.ImageLayoutUndefined, core1_0.ImageLayoutTransferDstOptimal, core1_0.PipelineStageTopOfPipe, core1_0.PipelineStageTransfer)
	if err != nil {
		log.Fatalln(err)
	}

	// Clear the input attachment image to yellow
	info.Cmd.CmdClearColorImage(inputImage, core1_0.ImageLayoutTransferDstOptimal, &common.ClearValueFloat{1, 1, 0, 0}, []common.ImageSubresourceRange{
		{
			AspectMask:     core1_0.AspectColor,
			BaseMipLevel:   0,
			LevelCount:     -1,
			BaseArrayLayer: 0,
			LayerCount:     -1,
		},
	})

	// Set the image layout to SHADER_READONLY_OPTIMAL for use by the shaders
	err = info.SetImageLayout(inputImage, core1_0.AspectColor, core1_0.ImageLayoutTransferDstOptimal, core1_0.ImageLayoutShaderReadOnlyOptimal, core1_0.PipelineStageTransfer, core1_0.PipelineStageFragmentShader)
	if err != nil {
		log.Fatalln(err)
	}

	inputAttachmentView, _, err := info.Loader.CreateImageView(info.Device, nil, core1_0.ImageViewCreateOptions{
		Image:    inputImage,
		ViewType: core1_0.ViewType2D,
		Format:   info.Format,
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
	if err != nil {
		log.Fatalln(err)
	}

	descLayout, _, err := info.Loader.CreateDescriptorSetLayout(info.Device, nil, core1_0.DescriptorSetLayoutCreateOptions{
		Bindings: []core1_0.DescriptorLayoutBinding{
			{
				Binding:         0,
				DescriptorType:  core1_0.DescriptorInputAttachment,
				DescriptorCount: 1,
				StageFlags:      core1_0.StageFragment,
			},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}
	info.DescLayout = []core1_0.DescriptorSetLayout{descLayout}

	info.PipelineLayout, _, err = info.Loader.CreatePipelineLayout(info.Device, nil, core1_0.PipelineLayoutCreateOptions{
		SetLayouts: info.DescLayout,
	})
	if err != nil {
		log.Fatalln(err)
	}

	attachments := []core1_0.AttachmentDescription{
		// First attachment is the color attachment - clear at the beginning of the
		// renderpass and transition layout to PRESENT_SRC_KHR at the end of
		// renderpass
		{
			Format:         info.Format,
			Samples:        core1_0.Samples1,
			LoadOp:         core1_0.LoadOpClear,
			StoreOp:        core1_0.StoreOpStore,
			StencilLoadOp:  core1_0.LoadOpDontCare,
			StencilStoreOp: core1_0.StoreOpDontCare,
			InitialLayout:  core1_0.ImageLayoutUndefined,
			FinalLayout:    khr_swapchain.ImageLayoutPresentSrc,
		},
		// Second attachment is input attachment.  Once cleared it should have
		// width*height yellow pixels.  Doing a subpassLoad in the fragment shader
		// should give the shader the color at the fragments x,y location
		// from the input attachment
		{
			Format:         info.Format,
			Samples:        core1_0.Samples1,
			LoadOp:         core1_0.LoadOpLoad,
			StoreOp:        core1_0.StoreOpDontCare,
			StencilLoadOp:  core1_0.LoadOpDontCare,
			StencilStoreOp: core1_0.StoreOpDontCare,
			InitialLayout:  core1_0.ImageLayoutShaderReadOnlyOptimal,
			FinalLayout:    core1_0.ImageLayoutShaderReadOnlyOptimal,
		},
	}

	colorRef := common.AttachmentReference{AttachmentIndex: 0, Layout: core1_0.ImageLayoutColorAttachmentOptimal}
	inputRef := common.AttachmentReference{AttachmentIndex: 1, Layout: core1_0.ImageLayoutShaderReadOnlyOptimal}

	subpass := core1_0.SubPass{
		BindPoint:        core1_0.BindGraphics,
		InputAttachments: []common.AttachmentReference{inputRef},
		ColorAttachments: []common.AttachmentReference{colorRef},
	}

	subpassDependency := core1_0.SubPassDependency{
		SrcSubPassIndex: core1_0.SubpassExternal,
		DstSubPassIndex: 0,
		SrcStageMask:    core1_0.PipelineStageColorAttachmentOutput,
		DstStageMask:    core1_0.PipelineStageColorAttachmentOutput,
		SrcAccessMask:   0,
		DstAccessMask:   core1_0.AccessColorAttachmentWrite,
	}

	info.RenderPass, _, err = info.Loader.CreateRenderPass(info.Device, nil, core1_0.RenderPassCreateOptions{
		Attachments:         attachments,
		SubPasses:           []core1_0.SubPass{subpass},
		SubPassDependencies: []core1_0.SubPassDependency{subpassDependency},
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
		framebuffer, _, err := info.Loader.CreateFrameBuffer(info.Device, nil, core1_0.FramebufferCreateOptions{
			RenderPass:  info.RenderPass,
			Attachments: []core1_0.ImageView{info.Buffers[i].View, inputAttachmentView},
			Width:       info.Width,
			Height:      info.Height,
			Layers:      1,
		})
		if err != nil {
			log.Fatalln(err)
		}
		info.Framebuffer = append(info.Framebuffer, framebuffer)
	}

	info.DescPool, _, err = info.Loader.CreateDescriptorPool(info.Device, nil, core1_0.DescriptorPoolCreateOptions{
		MaxSets: 1,
		PoolSizes: []core1_0.PoolSize{
			{
				Type:            core1_0.DescriptorInputAttachment,
				DescriptorCount: 1,
			},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	descSet, _, err := info.Loader.AllocateDescriptorSets(core1_0.DescriptorSetAllocateOptions{
		DescriptorPool:    info.DescPool,
		AllocationLayouts: []core1_0.DescriptorSetLayout{descLayout},
	})
	if err != nil {
		log.Fatalln(err)
	}

	info.DescSet = common.ConvertSlice(descSet, core.MapDescriptorSets)

	err = info.Device.UpdateDescriptorSets([]core1_0.WriteDescriptorSetOptions{
		{
			DstSet:          info.DescSet[0],
			DstBinding:      0,
			DstArrayElement: 0,
			DescriptorType:  core1_0.DescriptorInputAttachment,
			ImageInfo: []core1_0.DescriptorImageInfo{
				{
					ImageLayout: core1_0.ImageLayoutShaderReadOnlyOptimal,
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
	info.ImageAcquiredSemaphore, _, err = info.Loader.CreateSemaphore(info.Device, nil, core1_0.SemaphoreCreateOptions{})
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

	err = info.Cmd.CmdBeginRenderPass(core1_0.SubpassContentsInline, core1_0.RenderPassBeginOptions{
		RenderPass:  info.RenderPass,
		Framebuffer: info.Framebuffer[info.CurrentBuffer],
		RenderArea: common.Rect2D{
			Offset: common.Offset2D{X: 0, Y: 0},
			Extent: common.Extent2D{Width: info.Width, Height: info.Height},
		},
		ClearValues: []common.ClearValue{
			common.ClearValueFloat{0.2, 0.2, 0.2, 0.2},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}
	info.Cmd.CmdBindPipeline(core1_0.BindGraphics, info.Pipeline)
	info.Cmd.CmdBindDescriptorSets(core1_0.BindGraphics, info.PipelineLayout, info.DescSet, nil)

	info.InitViewports()
	info.InitScissors()

	info.Cmd.CmdDraw(3, 1, 0, 0)

	info.Cmd.CmdEndRenderPass()
	_, err = info.Cmd.End()
	if err != nil {
		log.Fatalln(err)
	}

	/* VULKAN_KEY_END */

	drawFence, _, err := info.Loader.CreateFence(info.Device, nil, core1_0.FenceCreateOptions{})
	if err != nil {
		log.Fatalln(err)
	}

	err = info.ExecuteQueueCmdBuf([]core1_0.CommandBuffer{info.Cmd}, drawFence)
	if err != nil {
		log.Fatalln(err)
	}

	for {
		res, err := drawFence.Wait(utils.FenceTimeout)
		if err != nil {
			log.Fatalln(err)
		}

		if res != core1_0.VKTimeout {
			break
		}
	}
	drawFence.Destroy(nil)

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

	info.ImageAcquiredSemaphore.Destroy(nil)
	inputAttachmentView.Destroy(nil)
	inputImage.Destroy(nil)
	inputMemory.Free(nil)
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
	info.Surface.Destroy(nil)
	debugMessenger.Destroy(nil)
	info.DestroyInstance()
	err = info.Window.Destroy()
	if err != nil {
		log.Fatalln(err)
	}
}
