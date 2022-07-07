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

func logDebug(msgType ext_debug_utils.MessageTypes, severity ext_debug_utils.MessageSeverities, data *ext_debug_utils.DebugUtilsMessengerCallbackData) bool {
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
	debugOptions := ext_debug_utils.DebugUtilsMessengerCreateInfo{
		MessageSeverity: ext_debug_utils.SeverityWarning | ext_debug_utils.SeverityError,
		MessageType:     ext_debug_utils.TypeGeneral | ext_debug_utils.TypeValidation | ext_debug_utils.TypePerformance,
		UserCallback:    logDebug,
	}

	err = info.InitInstance("Input Attachment Sample", debugOptions)
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

	props := info.Gpus[0].FormatProperties(core1_0.FormatR8G8B8A8UnsignedNormalized)
	if (props.OptimalTilingFeatures & core1_0.FormatFeatureColorAttachment) == 0 {
		log.Fatalf("%s format unsupported for input attachment\n", core1_0.FormatR8G8B8A8UnsignedNormalized)
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
	inputImage, _, err := info.Device.CreateImage(nil, core1_0.ImageCreateOptions{
		ImageType:     core1_0.ImageType2D,
		Format:        info.Format,
		Extent:        core1_0.Extent3D{Width: info.Width, Height: info.Height, Depth: 1},
		MipLevels:     1,
		ArrayLayers:   1,
		Samples:       utils.NumSamples,
		Tiling:        core1_0.ImageTilingOptimal,
		InitialLayout: core1_0.ImageLayoutUndefined,
		Usage:         core1_0.ImageUsageInputAttachment | core1_0.ImageUsageTransferDst,
		SharingMode:   core1_0.SharingModeExclusive,
	})
	if err != nil {
		log.Fatalln(err)
	}

	memReqs := inputImage.MemoryRequirements()
	memoryTypeIndex, err := info.MemoryTypeFromProperties(memReqs.MemoryTypeBits, 0)
	if err != nil {
		log.Fatalln(err)
	}

	inputMemory, _, err := info.Device.AllocateMemory(nil, core1_0.MemoryAllocateInfo{
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
	err = info.SetImageLayout(inputImage, core1_0.ImageAspectColor, core1_0.ImageLayoutUndefined, core1_0.ImageLayoutTransferDstOptimal, core1_0.PipelineStageTopOfPipe, core1_0.PipelineStageTransfer)
	if err != nil {
		log.Fatalln(err)
	}

	// Clear the input attachment image to yellow
	info.Cmd.CmdClearColorImage(inputImage, core1_0.ImageLayoutTransferDstOptimal, &core1_0.ClearValueFloat{1, 1, 0, 0}, []core1_0.ImageSubresourceRange{
		{
			AspectMask:     core1_0.ImageAspectColor,
			BaseMipLevel:   0,
			LevelCount:     -1,
			BaseArrayLayer: 0,
			LayerCount:     -1,
		},
	})

	// Set the image layout to SHADER_READONLY_OPTIMAL for use by the shaders
	err = info.SetImageLayout(inputImage, core1_0.ImageAspectColor, core1_0.ImageLayoutTransferDstOptimal, core1_0.ImageLayoutShaderReadOnlyOptimal, core1_0.PipelineStageTransfer, core1_0.PipelineStageFragmentShader)
	if err != nil {
		log.Fatalln(err)
	}

	inputAttachmentView, _, err := info.Device.CreateImageView(nil, core1_0.ImageViewCreateInfo{
		Image:    inputImage,
		ViewType: core1_0.ImageViewType2D,
		Format:   info.Format,
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
	if err != nil {
		log.Fatalln(err)
	}

	descLayout, _, err := info.Device.CreateDescriptorSetLayout(nil, core1_0.DescriptorSetLayoutCreateInfo{
		Bindings: []core1_0.DescriptorSetLayoutBinding{
			{
				Binding:         0,
				DescriptorType:  core1_0.DescriptorTypeInputAttachment,
				DescriptorCount: 1,
				StageFlags:      core1_0.StageFragment,
			},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}
	info.DescLayout = []core1_0.DescriptorSetLayout{descLayout}

	info.PipelineLayout, _, err = info.Device.CreatePipelineLayout(nil, core1_0.PipelineLayoutCreateInfo{
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
			LoadOp:         core1_0.AttachmentLoadOpClear,
			StoreOp:        core1_0.AttachmentStoreOpStore,
			StencilLoadOp:  core1_0.AttachmentLoadOpDontCare,
			StencilStoreOp: core1_0.AttachmentStoreOpDontCare,
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
			LoadOp:         core1_0.AttachmentLoadOpLoad,
			StoreOp:        core1_0.AttachmentStoreOpDontCare,
			StencilLoadOp:  core1_0.AttachmentLoadOpDontCare,
			StencilStoreOp: core1_0.AttachmentStoreOpDontCare,
			InitialLayout:  core1_0.ImageLayoutShaderReadOnlyOptimal,
			FinalLayout:    core1_0.ImageLayoutShaderReadOnlyOptimal,
		},
	}

	colorRef := core1_0.AttachmentReference{Attachment: 0, Layout: core1_0.ImageLayoutColorAttachmentOptimal}
	inputRef := core1_0.AttachmentReference{Attachment: 1, Layout: core1_0.ImageLayoutShaderReadOnlyOptimal}

	subpass := core1_0.SubpassDescription{
		PipelineBindPoint: core1_0.PipelineBindPointGraphics,
		InputAttachments:  []core1_0.AttachmentReference{inputRef},
		ColorAttachments:  []core1_0.AttachmentReference{colorRef},
	}

	subpassDependency := core1_0.SubpassDependency{
		SrcSubpass:    core1_0.SubpassExternal,
		DstSubpass:    0,
		SrcStageMask:  core1_0.PipelineStageColorAttachmentOutput,
		DstStageMask:  core1_0.PipelineStageColorAttachmentOutput,
		SrcAccessMask: 0,
		DstAccessMask: core1_0.AccessColorAttachmentWrite,
	}

	info.RenderPass, _, err = info.Device.CreateRenderPass(nil, core1_0.RenderPassCreateInfo{
		Attachments:         attachments,
		Subpasses:           []core1_0.SubpassDescription{subpass},
		SubpassDependencies: []core1_0.SubpassDependency{subpassDependency},
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
		framebuffer, _, err := info.Device.CreateFramebuffer(nil, core1_0.FramebufferCreateInfo{
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

	info.DescPool, _, err = info.Device.CreateDescriptorPool(nil, core1_0.DescriptorPoolCreateInfo{
		MaxSets: 1,
		PoolSizes: []core1_0.DescriptorPoolSize{
			{
				Type:            core1_0.DescriptorTypeInputAttachment,
				DescriptorCount: 1,
			},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	info.DescSet, _, err = info.Device.AllocateDescriptorSets(core1_0.DescriptorSetAllocateInfo{
		DescriptorPool: info.DescPool,
		SetLayouts:     []core1_0.DescriptorSetLayout{descLayout},
	})
	if err != nil {
		log.Fatalln(err)
	}

	err = info.Device.UpdateDescriptorSets([]core1_0.WriteDescriptorSet{
		{
			DstSet:          info.DescSet[0],
			DstBinding:      0,
			DstArrayElement: 0,
			DescriptorType:  core1_0.DescriptorTypeInputAttachment,
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
	info.ImageAcquiredSemaphore, _, err = info.Device.CreateSemaphore(nil, core1_0.SemaphoreCreateInfo{})
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

	err = info.Cmd.CmdBeginRenderPass(core1_0.SubpassContentsInline, core1_0.RenderPassBeginInfo{
		RenderPass:  info.RenderPass,
		Framebuffer: info.Framebuffer[info.CurrentBuffer],
		RenderArea: core1_0.Rect2D{
			Offset: core1_0.Offset2D{X: 0, Y: 0},
			Extent: core1_0.Extent2D{Width: info.Width, Height: info.Height},
		},
		ClearValues: []core1_0.ClearValue{
			core1_0.ClearValueFloat{0.2, 0.2, 0.2, 0.2},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}
	info.Cmd.CmdBindPipeline(core1_0.PipelineBindPointGraphics, info.Pipeline)
	info.Cmd.CmdBindDescriptorSets(core1_0.PipelineBindPointGraphics, info.PipelineLayout, info.DescSet, nil)

	info.InitViewports()
	info.InitScissors()

	info.Cmd.CmdDraw(3, 1, 0, 0)

	info.Cmd.CmdEndRenderPass()
	_, err = info.Cmd.End()
	if err != nil {
		log.Fatalln(err)
	}

	/* VULKAN_KEY_END */

	drawFence, _, err := info.Device.CreateFence(nil, core1_0.FenceCreateInfo{})
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
