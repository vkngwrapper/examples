package main

import (
	"bytes"
	"embed"
	"encoding/binary"
	"github.com/loov/hrtime"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/vkngwrapper/core/v3"
	"github.com/vkngwrapper/core/v3/common"
	"github.com/vkngwrapper/core/v3/core1_0"
	"github.com/vkngwrapper/examples/lunarg_samples/utils"
	"github.com/vkngwrapper/extensions/v3/ext_debug_utils"
	"github.com/vkngwrapper/extensions/v3/khr_swapchain"
	"log"
	"runtime/debug"
	"time"
	"unsafe"
)

//go:embed shaders
var fileSystem embed.FS

func logDebug(msgType ext_debug_utils.DebugUtilsMessageTypeFlags, severity ext_debug_utils.DebugUtilsMessageSeverityFlags, data *ext_debug_utils.DebugUtilsMessengerCallbackData) bool {
	log.Printf("[%s %s] - %s", severity, msgType, data.Message)

	if (severity & ext_debug_utils.SeverityError) != 0 {
		debug.PrintStack()
	}

	return false
}

/*
VULKAN_SAMPLE_SHORT_DESCRIPTION
Use a texel buffer to draw a magenta triangle
*/

func main() {
	texels := []float32{1, 0, 1}

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

	info.GlobalDriver, err = core.CreateDriverFromProcAddr(sdl.VulkanGetVkGetInstanceProcAddr())
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

	err = info.InitInstance("Texel Buffer Sample", debugOptions)
	if err != nil {
		log.Fatalln(err)
	}

	debugLoader := ext_debug_utils.CreateExtensionDriverFromCoreDriver(info.InstanceDriver)
	debugMessenger, _, err := debugLoader.CreateDebugUtilsMessenger(nil, debugOptions)
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitEnumerateDevice()
	if err != nil {
		log.Fatalln(err)
	}

	if info.GpuProps.Limits.MaxTexelBufferElements < 4 {
		log.Fatalln("maxTexelBufferElements too small")
	}

	props := info.InstanceDriver.GetPhysicalDeviceFormatProperties(info.Gpus[0], core1_0.FormatR32SignedFloat)
	if (props.BufferFeatures & core1_0.FormatFeatureUniformTexelBuffer) == 0 {
		log.Fatalln("R32_SFLOAT format unsupported for texel buffer")
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

	texelSize := binary.Size(texels)
	if texelSize < 0 {
		log.Fatalln("unsized texels")
	}

	texelBuf, _, err := info.DeviceDriver.CreateBuffer(nil, core1_0.BufferCreateInfo{
		Usage:       core1_0.BufferUsageUniformTexelBuffer,
		Size:        texelSize,
		SharingMode: core1_0.SharingModeExclusive,
	})
	if err != nil {
		log.Fatalln(err)
	}

	memReqs := info.DeviceDriver.GetBufferMemoryRequirements(texelBuf)

	memoryTypeIndex, err := info.MemoryTypeFromProperties(memReqs.MemoryTypeBits, core1_0.MemoryPropertyHostVisible|core1_0.MemoryPropertyHostCoherent)
	if err != nil {
		log.Fatalln(err)
	}

	texelMem, _, err := info.DeviceDriver.AllocateMemory(nil, core1_0.MemoryAllocateInfo{
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memoryTypeIndex,
	})
	if err != nil {
		log.Fatalln(err)
	}

	pData, _, err := info.DeviceDriver.MapMemory(texelMem, 0, memReqs.Size, 0)
	if err != nil {
		log.Fatalln(err)
	}

	memoryBytes := ([]byte)(unsafe.Slice((*byte)(pData), texelSize))
	writer := &bytes.Buffer{}
	err = binary.Write(writer, common.ByteOrder, texels)
	if err != nil {
		log.Fatalln(err)
	}
	copy(memoryBytes, writer.Bytes())

	info.DeviceDriver.UnmapMemory(texelMem)

	_, err = info.DeviceDriver.BindBufferMemory(texelBuf, texelMem, 0)
	if err != nil {
		log.Fatalln(err)
	}

	texelView, _, err := info.DeviceDriver.CreateBufferView(nil, core1_0.BufferViewCreateInfo{
		Buffer: texelBuf,
		Format: core1_0.FormatR32SignedFloat,
		Offset: 0,
		Range:  texelSize,
	})
	if err != nil {
		log.Fatalln(err)
	}

	/* Next take layout bindings and use them to create a descriptor set layout
	 */
	descLayout, _, err := info.DeviceDriver.CreateDescriptorSetLayout(nil, core1_0.DescriptorSetLayoutCreateInfo{
		Bindings: []core1_0.DescriptorSetLayoutBinding{
			{
				Binding:         0,
				DescriptorType:  core1_0.DescriptorTypeUniformTexelBuffer,
				DescriptorCount: 1,
				StageFlags:      core1_0.StageVertex,
			},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}
	info.DescLayout = append(info.DescLayout, descLayout)

	/* Now use the descriptor layout to create a pipeline layout */
	info.PipelineLayout, _, err = info.DeviceDriver.CreatePipelineLayout(nil, core1_0.PipelineLayoutCreateInfo{
		SetLayouts: info.DescLayout,
	})
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitRenderPass(false, true, khr_swapchain.ImageLayoutPresentSrc, core1_0.ImageLayoutUndefined)
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

	err = info.InitFramebuffers(false)
	if err != nil {
		log.Fatalln(err)
	}

	info.DescPool, _, err = info.DeviceDriver.CreateDescriptorPool(nil, core1_0.DescriptorPoolCreateInfo{
		MaxSets: 1,
		PoolSizes: []core1_0.DescriptorPoolSize{
			{
				Type:            core1_0.DescriptorTypeUniformTexelBuffer,
				DescriptorCount: 1,
			},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	/* Allocate descriptor set with UNIFORM_BUFFER_DYNAMIC */
	info.DescSet, _, err = info.DeviceDriver.AllocateDescriptorSets(core1_0.DescriptorSetAllocateInfo{
		DescriptorPool: info.DescPool,
		SetLayouts:     info.DescLayout,
	})
	if err != nil {
		log.Fatalln(err)
	}

	err = info.DeviceDriver.UpdateDescriptorSets([]core1_0.WriteDescriptorSet{
		{
			DstSet:          info.DescSet[0],
			DstBinding:      0,
			DstArrayElement: 0,

			DescriptorType:  core1_0.DescriptorTypeUniformTexelBuffer,
			TexelBufferView: []core1_0.BufferView{texelView},
		},
	}, nil)
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitPipelineCache()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitPipeline(false, false)
	if err != nil {
		log.Fatalln(err)
	}

	/* VULKAN_KEY_START */

	info.ImageAcquiredSemaphore, _, err = info.DeviceDriver.CreateSemaphore(nil, core1_0.SemaphoreCreateInfo{})
	if err != nil {
		log.Fatalln(err)
	}

	// Get the index of the next available swapchain image:
	info.CurrentBuffer, _, err = info.SwapchainExtension.AcquireNextImage(info.Swapchain, common.NoTimeout, &info.ImageAcquiredSemaphore, nil)
	if err != nil {
		log.Fatalln(err)
	}
	// TODO: Deal with the VK_SUBOPTIMAL_KHR and VK_ERROR_OUT_OF_DATE_KHR
	// return codes
	if err != nil {
		log.Fatalln(err)
	}

	err = info.DeviceDriver.CmdBeginRenderPass(info.Cmd, core1_0.SubpassContentsInline, core1_0.RenderPassBeginInfo{
		RenderPass:  info.RenderPass,
		Framebuffer: info.Framebuffer[info.CurrentBuffer],
		RenderArea: core1_0.Rect2D{
			Offset: core1_0.Offset2D{0, 0},
			Extent: core1_0.Extent2D{info.Width, info.Height},
		},
		ClearValues: []core1_0.ClearValue{
			core1_0.ClearValueFloat{0.2, 0.2, 0.2, 0.2},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	info.DeviceDriver.CmdBindPipeline(info.Cmd, core1_0.PipelineBindPointGraphics, info.Pipeline)
	info.DeviceDriver.CmdBindDescriptorSets(info.Cmd, core1_0.PipelineBindPointGraphics, info.PipelineLayout, 0, info.DescSet, nil)

	info.InitViewports()
	info.InitScissors()

	info.DeviceDriver.CmdDraw(info.Cmd, 3, 1, 0, 0)

	info.DeviceDriver.CmdEndRenderPass(info.Cmd)

	_, err = info.DeviceDriver.EndCommandBuffer(info.Cmd)
	if err != nil {
		log.Fatalln(err)
	}

	drawFence, _, err := info.DeviceDriver.CreateFence(nil, core1_0.FenceCreateInfo{})
	if err != nil {
		log.Fatalln(err)
	}

	err = info.ExecuteQueueCmdBuf([]core1_0.CommandBuffer{info.Cmd}, drawFence)
	if err != nil {
		log.Fatalln(err)
	}

	for {
		res, err := info.DeviceDriver.WaitForFences(true, utils.FenceTimeout, drawFence)
		if err != nil {
			log.Fatalln(err)
		}

		if res != core1_0.VKTimeout {
			break
		}
	}

	info.DeviceDriver.DestroyFence(drawFence, nil)

	err = info.ExecutePresentImage()
	if err != nil {
		log.Fatalln(err)
	}

	start := hrtime.Now()
	for hrtime.Since(start) < 5*time.Second {
		sdl.PollEvent()
	}

	/* VULKAN_KEY_END */
	if info.SaveImages {
		err = info.WritePNG("texel_buffer")
		if err != nil {
			log.Fatalln(err)
		}
	}

	info.DeviceDriver.DestroySemaphore(info.ImageAcquiredSemaphore, nil)
	info.DeviceDriver.DestroyBufferView(texelView, nil)
	info.DeviceDriver.DestroyBuffer(texelBuf, nil)
	info.DeviceDriver.FreeMemory(texelMem, nil)
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
		log.Fatal(err)
	}

	info.SurfaceDriver.DestroySurface(info.Surface, nil)
	debugLoader.DestroyDebugUtilsMessenger(debugMessenger, nil)
	info.DestroyInstance()
	err = info.Window.Destroy()
	if err != nil {
		log.Fatalln(err)
	}
}
