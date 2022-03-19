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
	"runtime/debug"
	"time"
	"unsafe"
)

//go:embed shaders
var fileSystem embed.FS

func logDebug(msgType ext_debug_utils.MessageTypes, severity ext_debug_utils.MessageSeverities, data *ext_debug_utils.CallbackDataOptions) bool {
	log.Printf("[%s %s] - %s\n", severity, msgType, data.Message)
	debug.PrintStack()
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
		CaptureTypes:      ext_debug_utils.TypeGeneral | ext_debug_utils.TypeValidation | ext_debug_utils.TypePerformance,
		Callback:          logDebug,
	}

	err = info.InitInstance("Texel Buffer Sample", debugOptions)
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

	if info.GpuProps.Limits.MaxTexelBufferElements < 4 {
		log.Fatalln("maxTexelBufferElements too small")
	}

	props := info.Gpus[0].FormatProperties(core1_0.DataFormatR32SignedFloat)
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

	texelBuf, _, err := info.Loader.CreateBuffer(info.Device, nil, &core1_0.BufferOptions{
		Usage:       core1_0.UsageUniformTexelBuffer,
		BufferSize:  texelSize,
		SharingMode: core1_0.SharingExclusive,
	})
	if err != nil {
		log.Fatalln(err)
	}

	memReqs := texelBuf.MemoryRequirements()

	memoryTypeIndex, err := info.MemoryTypeFromProperties(memReqs.MemoryType, core1_0.MemoryHostVisible|core1_0.MemoryHostCoherent)
	if err != nil {
		log.Fatalln(err)
	}

	texelMem, _, err := info.Device.AllocateMemory(nil, &core1_0.DeviceMemoryOptions{
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memoryTypeIndex,
	})
	if err != nil {
		log.Fatalln(err)
	}

	pData, _, err := texelMem.MapMemory(0, memReqs.Size, 0)
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

	texelMem.UnmapMemory()

	_, err = texelBuf.BindBufferMemory(texelMem, 0)
	if err != nil {
		log.Fatalln(err)
	}

	texelView, _, err := info.Loader.CreateBufferView(info.Device, nil, &core1_0.BufferViewOptions{
		Buffer: texelBuf,
		Format: core1_0.DataFormatR32SignedFloat,
		Offset: 0,
		Range:  texelSize,
	})
	if err != nil {
		log.Fatalln(err)
	}

	/* Next take layout bindings and use them to create a descriptor set layout
	 */
	descLayout, _, err := info.Loader.CreateDescriptorSetLayout(info.Device, nil, &core1_0.DescriptorSetLayoutOptions{
		Bindings: []core1_0.DescriptorLayoutBinding{
			{
				Binding:         0,
				DescriptorType:  core1_0.DescriptorUniformTexelBuffer,
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
	info.PipelineLayout, _, err = info.Loader.CreatePipelineLayout(info.Device, nil, &core1_0.PipelineLayoutOptions{
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

	info.DescPool, _, err = info.Loader.CreateDescriptorPool(info.Device, nil, &core1_0.DescriptorPoolOptions{
		MaxSets: 1,
		PoolSizes: []core1_0.PoolSize{
			{
				Type:            core1_0.DescriptorUniformTexelBuffer,
				DescriptorCount: 1,
			},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	/* Allocate descriptor set with UNIFORM_BUFFER_DYNAMIC */
	info.DescSet, _, err = info.Loader.AllocateDescriptorSets(&core1_0.DescriptorSetOptions{
		DescriptorPool:    info.DescPool,
		AllocationLayouts: info.DescLayout,
	})
	if err != nil {
		log.Fatalln(err)
	}

	err = info.Device.UpdateDescriptorSets([]core1_0.WriteDescriptorSetOptions{
		{
			DstSet:          info.DescSet[0],
			DstBinding:      0,
			DstArrayElement: 0,

			DescriptorType:  core1_0.DescriptorUniformTexelBuffer,
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

	info.ImageAcquiredSemaphore, _, err = info.Loader.CreateSemaphore(info.Device, nil, &core1_0.SemaphoreOptions{})
	if err != nil {
		log.Fatalln(err)
	}

	// Get the index of the next available swapchain image:
	info.CurrentBuffer, _, err = info.Swapchain.AcquireNextImage(common.NoTimeout, info.ImageAcquiredSemaphore, nil)
	if err != nil {
		log.Fatalln(err)
	}
	// TODO: Deal with the VK_SUBOPTIMAL_KHR and VK_ERROR_OUT_OF_DATE_KHR
	// return codes
	if err != nil {
		log.Fatalln(err)
	}

	err = info.Cmd.CmdBeginRenderPass(core1_0.SubpassContentsInline, &core1_0.RenderPassBeginOptions{
		RenderPass:  info.RenderPass,
		Framebuffer: info.Framebuffer[info.CurrentBuffer],
		RenderArea: common.Rect2D{
			Offset: common.Offset2D{0, 0},
			Extent: common.Extent2D{info.Width, info.Height},
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

	drawFence, _, err := info.Loader.CreateFence(info.Device, nil, &core1_0.FenceOptions{})
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

	/* VULKAN_KEY_END */
	if info.SaveImages {
		err = info.WritePNG("texel_buffer")
		if err != nil {
			log.Fatalln(err)
		}
	}

	info.ImageAcquiredSemaphore.Destroy(nil)
	texelView.Destroy(nil)
	texelBuf.Destroy(nil)
	texelMem.Free(nil)
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

	info.Surface.Destroy(nil)
	debugMessenger.Destroy(nil)
	info.DestroyInstance()
	err = info.Window.Destroy()
	if err != nil {
		log.Fatalln(err)
	}
}
