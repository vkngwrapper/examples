package main

import (
	"bytes"
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
	"time"
	"unsafe"
)

//go:embed shaders
var fileSystem embed.FS

func logDebug(msgType ext_debug_utils.MessageTypes, severity ext_debug_utils.MessageSeverities, data *ext_debug_utils.CallbackData) bool {
	log.Printf("[%s %s] - %s", severity, msgType, data.Message)
	return false
}

/*
VULKAN_SAMPLE_SHORT_DESCRIPTION
Use push constants in a simple shader, validate the correct value was read.
*/

// This sample submits two push constants and pairs them with a shader
// that simply reads in the values, ensures they are correct.  If correct
// values are read, the shader draws green.  If incorrect, shader draws red.

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

	err = info.InitInstance("Simple Push Constants", debugOptions)
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

	// Create binding and layout for the following, matching contents of shader
	//   binding 0 = uniform buffer (MVP)
	descriptorLayout, _, err := info.Loader.CreateDescriptorSetLayout(info.Device, nil, &core1_0.DescriptorSetLayoutOptions{
		Bindings: []core1_0.DescriptorLayoutBinding{
			{
				DescriptorType:  core1_0.DescriptorUniformBuffer,
				DescriptorCount: 1,
				Binding:         0,
				StageFlags:      core1_0.StageVertex,
			},
		},
	})

	/* VULKAN_KEY_START */

	// Set up our push constant range, which mirrors the declaration of
	pushConstantRanges := []core1_0.PushConstantRange{
		{
			Stages: core1_0.StageFragment,
			Offset: 0,
			Size:   8,
		},
	}

	// Create pipeline layout with multiple descriptor sets
	info.PipelineLayout, _, err = info.Loader.CreatePipelineLayout(info.Device, nil, &core1_0.PipelineLayoutOptions{
		PushConstantRanges: pushConstantRanges,
		SetLayouts:         []core1_0.DescriptorSetLayout{descriptorLayout},
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Create a single pool to contain data for our descriptor set
	descriptorPool, _, err := info.Loader.CreateDescriptorPool(info.Device, nil, &core1_0.DescriptorPoolOptions{
		MaxSets: 1,
		PoolSizes: []core1_0.PoolSize{
			{
				Type:            core1_0.DescriptorUniformBuffer,
				DescriptorCount: 1,
			},
			{
				Type:            core1_0.DescriptorCombinedImageSampler,
				DescriptorCount: 1,
			},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Populate descriptor sets
	descriptorSets, _, err := info.Loader.AllocateDescriptorSets(&core1_0.DescriptorSetOptions{
		DescriptorPool:    descriptorPool,
		AllocationLayouts: []core1_0.DescriptorSetLayout{descriptorLayout},
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Populate with info about our uniform buffer for MVP
	err = info.Device.UpdateDescriptorSets([]core1_0.WriteDescriptorSetOptions{
		{
			DstSet:          descriptorSets[0],
			DstBinding:      0,
			DstArrayElement: 0,

			DescriptorType: core1_0.DescriptorUniformBuffer,

			BufferInfo: []core1_0.DescriptorBufferInfo{
				info.UniformData.BufferInfo,
			},
		},
	}, nil)
	if err != nil {
		log.Fatalln(err)
	}

	// Create our push constant data, which matches shader expectations
	pushConstants := [2]uint32{2, 0x3F800000}
	pushConstantsSize := int(unsafe.Sizeof(pushConstants))

	// Ensure we have enough room for push constant data
	if pushConstantsSize > info.GpuProps.Limits.MaxPushConstantsSize {
		log.Fatalln("Too many push constants")
	}

	pushWriter := bytes.NewBuffer(make([]byte, 0, pushConstantsSize))
	err = binary.Write(pushWriter, common.ByteOrder, pushConstants)
	if err != nil {
		log.Fatalln(err)
	}

	info.Cmd.CmdPushConstants(info.PipelineLayout, core1_0.StageFragment, 0, pushWriter.Bytes())

	/* VULKAN_KEY_END */

	err = info.InitPipelineCache()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitPipeline(true, true)
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitPresentableImage()
	if err != nil {
		log.Fatalln(err)
	}

	clearValues := info.InitClearColorAndDepth()
	rpBegin := info.InitRenderPassBeginInfo()
	rpBegin.ClearValues = clearValues

	err = info.Cmd.CmdBeginRenderPass(core1_0.SubpassContentsInline, rpBegin)
	if err != nil {
		log.Fatal(err)
	}

	info.Cmd.CmdBindPipeline(core1_0.BindGraphics, info.Pipeline)
	info.Cmd.CmdBindDescriptorSets(core1_0.BindGraphics, info.PipelineLayout, descriptorSets, nil)
	info.Cmd.CmdBindVertexBuffers([]core1_0.Buffer{info.VertexBuffer.Buf}, []int{0})
	info.InitViewports()
	info.InitScissors()
	info.Cmd.CmdDraw(36, 1, 0, 0)
	info.Cmd.CmdEndRenderPass()
	_, err = info.Cmd.End()
	if err != nil {
		log.Fatalln(err)
	}

	drawFence, err := info.InitFence()
	if err != nil {
		log.Fatalln(err)
	}

	submitInfo := info.InitSubmitInfo(core1_0.PipelineStageColorAttachmentOutput)

	_, err = info.GraphicsQueue.SubmitToQueue(drawFence, []core1_0.SubmitOptions{*submitInfo})
	if err != nil {
		log.Fatalln(err)
	}

	presentInfo := info.InitPresentInfo()
	for {
		res, err := info.Device.WaitForFences(true, utils.FenceTimeout, []core1_0.Fence{drawFence})
		if err != nil {
			log.Fatalln(err)
		}

		if res != core1_0.VKTimeout {
			break
		}
	}

	_, err = info.SwapchainExtension.PresentToQueue(info.PresentQueue, presentInfo)
	if err != nil {
		log.Fatalln(err)
	}

	time.Sleep(time.Second)

	if info.SaveImages {
		err = info.WritePNG("push_constants")
		if err != nil {
			log.Fatalln(err)
		}
	}

	drawFence.Destroy(nil)
	info.ImageAcquiredSemaphore.Destroy(nil)
	info.DestroyPipeline()
	info.DestroyPipelineCache()

	descriptorPool.Destroy(nil)
	info.DestroyVertexBuffer()
	info.DestroyFramebuffers()
	info.DestroyShaders()
	info.DestroyRenderpass()

	descriptorLayout.Destroy(nil)
	info.PipelineLayout.Destroy(nil)
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
