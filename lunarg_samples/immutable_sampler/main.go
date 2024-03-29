package main

import (
	"embed"
	"encoding/binary"
	"github.com/loov/hrtime"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/vkngwrapper/core/v2"
	"github.com/vkngwrapper/core/v2/core1_0"
	"github.com/vkngwrapper/examples/lunarg_samples/utils"
	"github.com/vkngwrapper/extensions/v2/ext_debug_utils"
	"github.com/vkngwrapper/extensions/v2/khr_swapchain"
	"log"
	"runtime/debug"
	"time"
	"unsafe"
)

//go:embed shaders images
var fileSystem embed.FS

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
Use an immutable sampler to texture a cube.
*/

// This sample is based on template and uses an immutable sampler,
// along with a sampled image.  It should render the LunarG textured cube.

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

	err = info.InitInstance("Simple Immutable Sampler", debugOptions)
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
	textureObj, err := info.InitImage(imageFile, 0, 0)
	if err != nil {
		log.Fatalln(err)
	}

	info.Textures = append(info.Textures, textureObj)

	info.TextureData.ImageInfo.ImageView = textureObj.View
	info.TextureData.ImageInfo.ImageLayout = core1_0.ImageLayoutShaderReadOnlyOptimal

	// Set up one descriptor sets
	const descriptorSetCount = 1

	// Create binding and layout for the following, matching contents of shader
	//   binding 0 = uniform buffer (MVP)
	//   binding 1 = combined image and immutable sampler
	resourceBinding := []core1_0.DescriptorSetLayoutBinding{
		{
			Binding:         0,
			DescriptorType:  core1_0.DescriptorTypeUniformBuffer,
			DescriptorCount: 1,
			StageFlags:      core1_0.StageVertex,
		},
		{
			Binding:           1,
			DescriptorType:    core1_0.DescriptorTypeCombinedImageSampler,
			DescriptorCount:   1,
			StageFlags:        core1_0.StageFragment,
			ImmutableSamplers: []core1_0.Sampler{immutableSampler},
		},
	}

	descriptorLayout, _, err := info.Device.CreateDescriptorSetLayout(nil, core1_0.DescriptorSetLayoutCreateInfo{
		Bindings: resourceBinding,
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Create pipeline layout
	info.PipelineLayout, _, err = info.Device.CreatePipelineLayout(nil, core1_0.PipelineLayoutCreateInfo{
		SetLayouts: []core1_0.DescriptorSetLayout{descriptorLayout},
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Create a single pool to contain data for our descriptor set
	poolSizes := []core1_0.DescriptorPoolSize{
		{
			Type:            core1_0.DescriptorTypeUniformBuffer,
			DescriptorCount: 1,
		},
		{
			Type:            core1_0.DescriptorTypeCombinedImageSampler,
			DescriptorCount: 1,
		},
	}

	descriptorPool, _, err := info.Device.CreateDescriptorPool(nil, core1_0.DescriptorPoolCreateInfo{
		MaxSets:   descriptorSetCount,
		PoolSizes: poolSizes,
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Populate descriptor sets
	descriptorSets, _, err := info.Device.AllocateDescriptorSets(core1_0.DescriptorSetAllocateInfo{
		DescriptorPool: descriptorPool,
		SetLayouts:     []core1_0.DescriptorSetLayout{descriptorLayout},
	})
	if err != nil {
		log.Fatalln(err)
	}

	err = info.Device.UpdateDescriptorSets([]core1_0.WriteDescriptorSet{
		{
			DstSet:          descriptorSets[0],
			DstBinding:      0,
			DstArrayElement: 0,
			DescriptorType:  core1_0.DescriptorTypeUniformBuffer,
			BufferInfo:      []core1_0.DescriptorBufferInfo{info.UniformData.BufferInfo},
		},
		{
			DstSet:          descriptorSets[0],
			DstBinding:      1,
			DstArrayElement: 0,
			DescriptorType:  core1_0.DescriptorTypeCombinedImageSampler,
			ImageInfo:       []core1_0.DescriptorImageInfo{info.TextureData.ImageInfo},
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
		log.Fatalln(err)
	}

	info.Cmd.CmdBindPipeline(core1_0.PipelineBindPointGraphics, info.Pipeline)
	info.Cmd.CmdBindDescriptorSets(core1_0.PipelineBindPointGraphics, info.PipelineLayout, 0, descriptorSets, nil)

	info.Cmd.CmdBindVertexBuffers(0, []core1_0.Buffer{info.VertexBuffer.Buf}, []int{0})

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

	_, err = info.GraphicsQueue.Submit(drawFence, []core1_0.SubmitInfo{*submitInfo})
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
		if res != core1_0.VKTimeout {
			break
		}
	}

	_, err = info.SwapchainExtension.QueuePresent(info.PresentQueue, present)
	if err != nil {
		log.Fatalln(err)
	}

	start := hrtime.Now()
	for hrtime.Since(start) < 5*time.Second {
		sdl.PollEvent()
	}

	if info.SaveImages {
		err = info.WritePNG("immutable_sampler")
		if err != nil {
			log.Fatalln(err)
		}
	}

	drawFence.Destroy(nil)
	info.ImageAcquiredSemaphore.Destroy(nil)
	info.DestroyPipeline()
	info.DestroyPipelineCache()

	immutableSampler.Destroy(nil)
	info.Textures[0].View.Destroy(nil)
	info.Textures[0].Image.Destroy(nil)
	info.Textures[0].ImageMemory.Free(nil)

	if info.Textures[0].NeedsStaging {
		info.Textures[0].Buffer.Destroy(nil)
		info.Textures[0].BufferMemory.Free(nil)
	}

	// instead of destroy_descriptor_pool(info);
	descriptorPool.Destroy(nil)

	info.DestroyVertexBuffer()
	info.DestroyFramebuffers()
	info.DestroyShaders()
	info.DestroyRenderpass()

	// instead of destroy_descriptor_and_pipeline_layouts(info);
	descriptorLayout.Destroy(nil)
	info.PipelineLayout.Destroy(nil)

	info.DestroyUniformBuffer()
	info.DestroyDepthBuffer()
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
