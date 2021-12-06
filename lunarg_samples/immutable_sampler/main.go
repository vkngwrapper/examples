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
	"runtime/debug"
	"time"
	"unsafe"
)

//go:embed shaders images
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

	err = info.InitInstance("Simple Immutable Sampler", debugOptions)
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

	/* VULKAN_KEY_START */

	// Create the sampler we'll be using immutably
	immutableSampler, err := info.InitSampler()
	if err != nil {
		log.Fatalln(err)
	}

	// Call helper that inits image without attaching sampler
	imageFile, err := fileSystem.Open("images/lunarg.png")
	if err != nil {
		log.Fatalln(err)
	}
	textureObj, err := info.InitImage(imageFile)
	if err != nil {
		log.Fatalln(err)
	}

	info.Textures = append(info.Textures, textureObj)

	info.TextureData.ImageInfo.ImageView = textureObj.View
	info.TextureData.ImageInfo.ImageLayout = common.LayoutShaderReadOnlyOptimal

	// Set up one descriptor sets
	const descriptorSetCount = 1
	const resourceCount = 2
	const resourceTypeCount = 2

	// Create binding and layout for the following, matching contents of shader
	//   binding 0 = uniform buffer (MVP)
	//   binding 1 = combined image and immutable sampler
	resourceBinding := []*core.DescriptorLayoutBinding{
		{
			Binding:      0,
			Type:         common.DescriptorUniformBuffer,
			Count:        1,
			ShaderStages: common.StageVertex,
		},
		{
			Binding:           1,
			Type:              common.DescriptorCombinedImageSampler,
			Count:             1,
			ShaderStages:      common.StageFragment,
			ImmutableSamplers: []core.Sampler{immutableSampler},
		},
	}

	descriptorLayout, _, err := info.Loader.CreateDescriptorSetLayout(info.Device, &core.DescriptorSetLayoutOptions{
		Bindings: resourceBinding,
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Create pipeline layout
	info.PipelineLayout, _, err = info.Loader.CreatePipelineLayout(info.Device, &core.PipelineLayoutOptions{
		SetLayouts: []core.DescriptorSetLayout{descriptorLayout},
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Create a single pool to contain data for our descriptor set
	poolSizes := []core.PoolSize{
		{
			Type:  common.DescriptorUniformBuffer,
			Count: 1,
		},
		{
			Type:  common.DescriptorCombinedImageSampler,
			Count: 1,
		},
	}

	descriptorPool, _, err := info.Loader.CreateDescriptorPool(info.Device, &core.DescriptorPoolOptions{
		MaxSets:   descriptorSetCount,
		PoolSizes: poolSizes,
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

	err = info.Device.UpdateDescriptorSets([]core.WriteDescriptorSetOptions{
		{
			Destination:             descriptorSets[0],
			DestinationBinding:      0,
			DestinationArrayElement: 0,
			DescriptorType:          common.DescriptorUniformBuffer,
			BufferInfo:              []core.DescriptorBufferInfo{info.UniformData.BufferInfo},
		},
		{
			Destination:             descriptorSets[0],
			DestinationBinding:      1,
			DestinationArrayElement: 0,
			DescriptorType:          common.DescriptorCombinedImageSampler,
			ImageInfo:               []core.DescriptorImageInfo{info.TextureData.ImageInfo},
		},
	}, nil)
	if err != nil {
		log.Fatalln(err)
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
		log.Fatalln(err)
	}

	info.Cmd.CmdBindPipeline(common.BindGraphics, info.Pipeline)
	info.Cmd.CmdBindDescriptorSets(common.BindGraphics, info.PipelineLayout, 0, descriptorSets, nil)

	info.Cmd.CmdBindVertexBuffers(0, []core.Buffer{info.VertexBuffer.Buf}, []int{0})

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
	submitInfo := info.InitSubmitInfo(common.PipelineStageColorAttachmentOutput)

	_, err = info.GraphicsQueue.SubmitToQueue(drawFence, []*core.SubmitOptions{submitInfo})
	if err != nil {
		log.Fatalln(err)
	}

	/* Now present the image in the window */
	present := info.InitPresentInfo()

	/* Make sure command buffer is finished before presenting */
	for {
		res, err := drawFence.Wait(utils.FenceTimeout)
		if err != nil {
			log.Fatalln(err)
		}
		if res != core.VKTimeout {
			break
		}
	}

	_, _, err = info.Swapchain.PresentToQueue(info.PresentQueue, present)
	if err != nil {
		log.Fatalln(err)
	}

	time.Sleep(time.Second)

	if info.SaveImages {
		err = info.WritePNG("immutable_sampler")
		if err != nil {
			log.Fatalln(err)
		}
	}

	drawFence.Destroy()
	info.ImageAcquiredSemaphore.Destroy()
	info.DestroyPipeline()
	info.DestroyPipelineCache()

	immutableSampler.Destroy()
	info.Textures[0].View.Destroy()
	info.Textures[0].Image.Destroy()
	info.Device.FreeMemory(info.Textures[0].ImageMemory)

	if info.Textures[0].NeedsStaging {
		info.Textures[0].Buffer.Destroy()
		info.Device.FreeMemory(info.Textures[0].BufferMemory)
	}

	// instead of destroy_descriptor_pool(info);
	descriptorPool.Destroy()

	info.DestroyVertexBuffer()
	info.DestroyFramebuffers()
	info.DestroyShaders()
	info.DestroyRenderpass()

	// instead of destroy_descriptor_and_pipeline_layouts(info);
	descriptorLayout.Destroy()
	info.PipelineLayout.Destroy()

	info.DestroyUniformBuffer()
	info.DestroyDepthBuffer()
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
	info.Window.Destroy()
}
