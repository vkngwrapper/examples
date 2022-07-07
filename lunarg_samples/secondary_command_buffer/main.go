package main

import (
	"embed"
	"encoding/binary"
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
	"unsafe"
)

//go:embed shaders images
var fileSystem embed.FS

func logDebug(msgType ext_debug_utils.MessageTypes, severity ext_debug_utils.MessageSeverities, data *ext_debug_utils.DebugUtilsMessengerCallbackData) bool {
	log.Printf("[%s %s] - %s\n", severity, msgType, data.Message)
	debug.PrintStack()
	return false
}

/*
VULKAN_SAMPLE_SHORT_DESCRIPTION
Draw several cubes using primary and secondary command buffers
*/

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

	err = info.InitInstance("Secondary Command Buffers", debugOptions)
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

	err = info.InitDepthBuffer()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitUniformBuffer()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitDescriptorAndPipelineLayouts(true)
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitRenderPass(true, true, khr_swapchain.ImageLayoutPresentSrc, core1_0.ImageLayoutUndefined)
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

	err = info.InitFramebuffers(true)
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitVertexBuffers(utils.VBTextureData, binary.Size(utils.VBTextureData), int(unsafe.Sizeof(utils.VertexUV{})), true)
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitPipelineCache()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitPipeline(true, true)
	if err != nil {
		log.Fatalln(err)
	}

	// we have to set up a couple of things by hand, but this
	// isn't any different to other examples

	// get two different textures
	imageFile, err := fileSystem.Open("images/green.png")
	if err != nil {
		log.Fatalln(err)
	}
	err = info.InitTexture(imageFile, 0, 0)
	if err != nil {
		log.Fatalln(err)
	}
	greenTex := info.TextureData.ImageInfo

	imageFile, err = fileSystem.Open("images/lunarg.png")
	if err != nil {
		log.Fatalln(err)
	}
	err = info.InitTexture(imageFile, 0, 0)
	if err != nil {
		log.Fatalln(err)
	}
	lunargTex := info.TextureData.ImageInfo

	// create two identical descriptor sets, each with a different texture but
	// identical UBOa
	info.DescPool, _, err = info.Device.CreateDescriptorPool(nil, core1_0.DescriptorPoolCreateInfo{
		PoolSizes: []core1_0.DescriptorPoolSize{
			{
				Type:            core1_0.DescriptorTypeUniformBuffer,
				DescriptorCount: 2,
			},
			{
				Type:            core1_0.DescriptorTypeCombinedImageSampler,
				DescriptorCount: 2,
			},
		},
		MaxSets: 2,
	})
	if err != nil {
		log.Fatalln(err)
	}

	info.DescSet, _, err = info.Device.AllocateDescriptorSets(core1_0.DescriptorSetAllocateInfo{
		DescriptorPool: info.DescPool,
		SetLayouts:     []core1_0.DescriptorSetLayout{info.DescLayout[0], info.DescLayout[0]},
	})
	if err != nil {
		log.Fatalln(err)
	}

	writes := []core1_0.WriteDescriptorSet{
		{
			DstSet:          info.DescSet[0],
			DstBinding:      0,
			DstArrayElement: 0,

			DescriptorType: core1_0.DescriptorTypeUniformBuffer,
			BufferInfo:     []core1_0.DescriptorBufferInfo{info.UniformData.BufferInfo},
		},
		{
			DstSet:          info.DescSet[0],
			DstBinding:      1,
			DstArrayElement: 0,

			DescriptorType: core1_0.DescriptorTypeCombinedImageSampler,
			ImageInfo:      []core1_0.DescriptorImageInfo{greenTex},
		},
	}
	err = info.Device.UpdateDescriptorSets(writes, nil)
	if err != nil {
		log.Fatalln(err)
	}

	writes[0].DstSet = info.DescSet[1]
	writes[1].DstSet = info.DescSet[1]
	writes[1].ImageInfo[0] = lunargTex
	err = info.Device.UpdateDescriptorSets(writes, nil)
	if err != nil {
		log.Fatalln(err)
	}

	/* VULKAN_KEY_START */

	// create four secondary command buffers, for each quadrant of the screen
	secondaryCmds, _, err := info.Device.AllocateCommandBuffers(core1_0.CommandBufferAllocateInfo{
		CommandPool:        info.CmdPool,
		Level:              core1_0.CommandBufferLevelSecondary,
		CommandBufferCount: 4,
	})
	if err != nil {
		log.Fatalln(err)
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

	err = info.SetImageLayout(info.Buffers[info.CurrentBuffer].Image, core1_0.ImageAspectColor, core1_0.ImageLayoutUndefined, core1_0.ImageLayoutColorAttachmentOptimal, core1_0.PipelineStageTopOfPipe, core1_0.PipelineStageColorAttachmentOutput)
	if err != nil {
		log.Fatalln(err)
	}

	viewport := core1_0.Viewport{
		X: 0, Y: 0,
		MinDepth: 0, MaxDepth: 1,
		Width: 200, Height: 200,
	}

	scissor := core1_0.Rect2D{
		Offset: core1_0.Offset2D{0, 0},
		Extent: core1_0.Extent2D{info.Width, info.Height},
	}

	// now we record four separate command buffers, one for each quadrant of the
	// screen
	inheritanceInfo := &core1_0.CommandBufferInheritanceInfo{
		Framebuffer: info.Framebuffer[info.CurrentBuffer],
		RenderPass:  info.RenderPass,
		Subpass:     0,
	}
	secondaryBegin := core1_0.CommandBufferBeginInfo{
		Flags:           core1_0.CommandBufferUsageOneTimeSubmit | core1_0.CommandBufferUsageRenderPassContinue,
		InheritanceInfo: inheritanceInfo,
	}

	for i := 0; i < 4; i++ {
		_, err = secondaryCmds[i].Begin(secondaryBegin)
		if err != nil {
			log.Fatalln(err)
		}

		secondaryCmds[i].CmdBindPipeline(core1_0.PipelineBindPointGraphics, info.Pipeline)
		firstIndex := 0
		secondIndex := 1

		if i == 0 || i == 3 {
			firstIndex = 1
			secondIndex = 2
		}
		secondaryCmds[i].CmdBindDescriptorSets(core1_0.PipelineBindPointGraphics, info.PipelineLayout, info.DescSet[firstIndex:secondIndex], nil)
		secondaryCmds[i].CmdBindVertexBuffers([]core1_0.Buffer{info.VertexBuffer.Buf}, []int{0})

		viewport.X = 25.0 + 250.0*float32(i%2)
		viewport.Y = 25.0 + 250.0*float32(i/2)
		secondaryCmds[i].CmdSetViewport([]core1_0.Viewport{viewport})
		secondaryCmds[i].CmdSetScissor([]core1_0.Rect2D{scissor})

		secondaryCmds[i].CmdDraw(36, 1, 0, 0)
		_, err = secondaryCmds[i].End()
		if err != nil {
			log.Fatalln(err)
		}
	}

	// specifying VK_SUBPASS_CONTENTS_SECONDARY_COMMAND_BUFFERS means this
	// render pass may
	// ONLY call vkCmdExecuteCommands
	err = info.Cmd.CmdBeginRenderPass(core1_0.SubpassContentsSecondaryCommandBuffers, core1_0.RenderPassBeginInfo{
		RenderPass:  info.RenderPass,
		Framebuffer: info.Framebuffer[info.CurrentBuffer],
		RenderArea: core1_0.Rect2D{
			Offset: core1_0.Offset2D{0, 0},
			Extent: core1_0.Extent2D{info.Width, info.Height},
		},
		ClearValues: []core1_0.ClearValue{
			core1_0.ClearValueFloat{0.2, 0.2, 0.2, 0.2},
			core1_0.ClearValueDepthStencil{Depth: 1, Stencil: 0},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	info.Cmd.CmdExecuteCommands(secondaryCmds)

	info.Cmd.CmdEndRenderPass()

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
			CommandBuffers:   []core1_0.CommandBuffer{info.Cmd},
			WaitSemaphores:   []core1_0.Semaphore{imageAcquiredSemaphore},
			WaitDstStageMask: []core1_0.PipelineStageFlags{core1_0.PipelineStageColorAttachmentOutput},
		},
	})
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

	time.Sleep(time.Second)

	if info.SaveImages {
		err = info.WritePNG("secondary_command_buffer")
		if err != nil {
			log.Fatalln(err)
		}
	}

	info.Device.FreeCommandBuffers(secondaryCmds)

	/* VULKAN_KEY_END */

	drawFence.Destroy(nil)
	imageAcquiredSemaphore.Destroy(nil)
	info.DestroyPipeline()
	info.DestroyPipelineCache()
	info.DestroyTextures()
	info.DestroyDescriptorPool()
	info.DestroyVertexBuffer()
	info.DestroyFramebuffers()
	info.DestroyShaders()
	info.DestroyRenderpass()
	info.DestroyDescriptorAndPipelineLayouts()
	info.DestroyUniformBuffer()
	info.DestroyDepthBuffer()
	info.DestroySwapchain()
	info.DestroyCommandBuffer()
	info.DestroyCommandPool()

	err = info.DestroyDevice()
	if err != nil {
		log.Fatal(err)
	}

	info.Surface.Destroy(nil)
	debugMessenger.Destroy(nil)
	info.DestroyInstance()
	err = info.Window.Destroy()
	if err != nil {
		log.Fatalln(err)
	}
}
