package main

import (
	"embed"
	"encoding/binary"
	"github.com/CannibalVox/VKng/core"
	"github.com/CannibalVox/VKng/core/common"
	"github.com/CannibalVox/VKng/examples/lunarg_samples/utils"
	"github.com/CannibalVox/VKng/extensions/ext_debug_utils"
	"github.com/veandco/go-sdl2/sdl"
	"log"
	"time"
	"unsafe"
)

//go:embed shaders
var fileSystem embed.FS

func logDebug(msgType ext_debug_utils.MessageType, severity ext_debug_utils.MessageSeverity, data *ext_debug_utils.CallbackData) bool {
	log.Printf("[%s %s] - %s", severity, msgType, data.Message)
	return false
}

func main() {
	info := &utils.SampleInfo{}
	err := utils.ProcessCommandLineArgs(info)
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitWindowSize(1000, 1000)
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

	debugLoader := ext_debug_utils.CreateLoaderFromInstance(info.Instance)
	debugMessenger, _, err := debugLoader.CreateMessenger(info.Instance, debugOptions)
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

	err = info.InitSwapchain()
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

	err = info.InitRenderPass(true)
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
	descriptorLayout, _, err := info.Loader.CreateDescriptorSetLayout(info.Device, &core.DescriptorSetLayoutOptions{
		Bindings: []*core.DescriptorLayoutBinding{
			{
				Type:         common.DescriptorUniformBuffer,
				Count:        1,
				Binding:      0,
				ShaderStages: common.StageVertex,
			},
		},
	})

	/* VULKAN_KEY_START */

	// Set up our push constant range, which mirrors the declaration of
	pushConstantRanges := []common.PushConstantRange{
		{
			Stages: common.StageFragment,
			Offset: 0,
			Size:   8,
		},
	}

	// Create pipeline layout with multiple descriptor sets
	info.PipelineLayout, _, err = info.Loader.CreatePipelineLayout(info.Device, &core.PipelineLayoutOptions{
		PushConstantRanges: pushConstantRanges,
		SetLayouts:         []core.DescriptorSetLayout{descriptorLayout},
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Create a single pool to contain data for our descriptor set
	descriptorPool, _, err := info.Loader.CreateDescriptorPool(info.Device, &core.DescriptorPoolOptions{
		MaxSets: 1,
		PoolSizes: []core.PoolSize{
			{
				Type:  common.DescriptorUniformBuffer,
				Count: 1,
			},
			{
				Type:  common.DescriptorCombinedImageSampler,
				Count: 1,
			},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Populate descriptor sets
	descriptorSets, _, err := descriptorPool.AllocateDescriptorSets(&core.DescriptorSetOptions{
		AllocationLayouts: []core.DescriptorSetLayout{descriptorLayout},
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Populate with info about our uniform buffer for MVP
	err = info.Device.UpdateDescriptorSets([]core.WriteDescriptorSetOptions{
		{
			Destination:             descriptorSets[0],
			DestinationBinding:      0,
			DestinationArrayElement: 0,

			DescriptorType: common.DescriptorUniformBuffer,

			BufferInfo: []core.DescriptorBufferInfo{
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

	err = info.Cmd.CmdPushConstants(info.PipelineLayout, common.StageFragment, 0, pushConstants)
	if err != nil {
		log.Fatalln()
	}

	/* VULKAN_KEY_END */

	err = info.InitPipelineCache()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitPipeline(true)
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

	err = info.Cmd.CmdBeginRenderPass(core.ContentsInline, rpBegin)
	if err != nil {
		log.Fatal(err)
	}

	err = info.Cmd.CmdBindPipeline(common.BindGraphics, info.Pipeline)
	if err != nil {
		log.Fatalln(err)
	}

	err = info.Cmd.CmdBindDescriptorSets(common.BindGraphics, info.PipelineLayout, 0, descriptorSets, nil)
	if err != nil {
		log.Fatalln(err)
	}

	err = info.Cmd.CmdBindVertexBuffers(0, []core.Buffer{info.VertexBuffer.Buf}, []int{0})
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitViewports()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitScissors()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.Cmd.CmdDraw(36, 1, 0, 0)
	if err != nil {
		log.Fatalln(err)
	}

	err = info.Cmd.CmdEndRenderPass()
	if err != nil {
		log.Fatalln(err)
	}

	_, err = info.Cmd.End()
	if err != nil {
		log.Fatalln(err)
	}

	drawFence, err := info.InitFence()
	if err != nil {
		log.Fatalln(err)
	}

	submitInfo := info.InitSubmitInfo(common.PipelineStageColorAttachmentOutput)

	_, err = info.GraphicsQueue.SubmitToQueue(drawFence, []*core.SubmitOptions{submitInfo})
	if err != nil {
		log.Fatalln(err)
	}

	presentInfo := info.InitPresentInfo()
	for {
		res, err := info.Device.WaitForFences(true, utils.FenceTimeout, []core.Fence{drawFence})
		if err != nil {
			log.Fatalln(err)
		}

		if res != core.VKTimeout {
			break
		}
	}

	_, _, err = info.Swapchain.PresentToQueue(info.PresentQueue, presentInfo)
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

	drawFence.Destroy()
	info.ImageAcquiredSemaphore.Destroy()
	info.DestroyPipeline()
	info.DestroyPipelineCache()

	descriptorPool.Destroy()
	info.DestroyVertexBuffer()
	info.DestroyFramebuffers()
	info.DestroyShaders()
	info.DestroyRenderpass()

	descriptorLayout.Destroy()
	info.PipelineLayout.Destroy()
	info.DestroyUniformBuffer()
	info.DestroyDepthBuffer()
	info.DestroySwapchain()
	info.DestroyCommandBuffer()
	info.DestroyCommandPool()

	err = info.DestroyDevice()
	if err != nil {
		log.Fatal(err)
	}

	info.Surface.Destroy()
	debugMessenger.Destroy()
	info.DestroyInstance()
	info.Window.Destroy()
}
