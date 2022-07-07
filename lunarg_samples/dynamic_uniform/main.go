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
	"github.com/go-gl/mathgl/mgl32"
	"github.com/veandco/go-sdl2/sdl"
	"log"
	"runtime/debug"
	"time"
	"unsafe"
)

//go:embed shaders
var fileSystem embed.FS

func logDebug(msgType ext_debug_utils.MessageTypes, severity ext_debug_utils.MessageSeverities, data *ext_debug_utils.DebugUtilsMessengerCallbackData) bool {
	log.Printf("[%s %s] - %s", severity, msgType, data.Message)
	debug.PrintStack()
	log.Println()
	return false
}

/*
VULKAN_SAMPLE_SHORT_DESCRIPTION
Draw 2 Cubes using dynamic uniform buffer
*/
/* This sample builds upon the drawcube sample by using a dynamic uniform */
/* buffer to store two transformation matrices, using the first matrix on */
/* the first draw, and then specifying an offset to the second matrix in  */
/* the buffer for the second draw, resulting in 2 cubes offset from each  */
/* other                                                                  */
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

	err = info.InitInstance("Draw Cube", debugOptions)
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

	if info.GpuProps.Limits.MaxDescriptorSetUniformBuffersDynamic < 1 {
		log.Fatalln("No dynamic uniform buffers supported")
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

	err = info.InitVertexBuffers(utils.VBSolidFaceColorsData, binary.Size(utils.VBSolidFaceColorsData), int(unsafe.Sizeof(utils.Vertex{})), false)
	if err != nil {
		log.Fatalln(err)
	}

	/* Set up uniform buffer with 2 transform matrices in it */
	info.Projection = mgl32.Perspective(mgl32.DegToRad(45), 1, 0.1, 100)
	info.View = mgl32.LookAt(0, 3, -10, 0, 0, 0, 0, -1, 0)
	info.Model = mgl32.Ident4()
	// Vulkan clip space has inverted Y and half Z.
	info.Clip = mgl32.Mat4{1, 0, 0, 0, 0, -1, 0, 0, 0, 0, 0.5, 0, 0, 0, 0.5, 1}
	info.MVP = info.Clip.Mul4(info.Projection).Mul4(info.View).Mul4(info.Model)

	/* VULKAN_KEY_START */
	info.Model = info.Model.Mul4(mgl32.Translate3D(-1.5, 1.5, -1.5))
	mvp2 := info.Clip.Mul4(info.Projection).Mul4(info.View).Mul4(info.Model)
	bufSize := int(unsafe.Sizeof(info.MVP))

	if info.GpuProps.Limits.MinUniformBufferOffsetAlignment != 0 {
		bufSize = (bufSize + info.GpuProps.Limits.MinUniformBufferOffsetAlignment - 1) &
			^(info.GpuProps.Limits.MinUniformBufferOffsetAlignment - 1)
	}

	info.UniformData.Buf, _, err = info.Device.CreateBuffer(nil, core1_0.BufferCreateInfo{
		Usage:       core1_0.BufferUsageUniformBuffer,
		Size:        2 * bufSize,
		SharingMode: core1_0.SharingModeExclusive,
	})
	if err != nil {
		log.Fatalln(err)
	}

	memReqs := info.UniformData.Buf.MemoryRequirements()

	memoryTypeIndex, err := info.MemoryTypeFromProperties(memReqs.MemoryTypeBits, core1_0.MemoryPropertyHostVisible|core1_0.MemoryPropertyHostCoherent)
	if err != nil {
		log.Fatalln(err)
	}

	info.UniformData.Mem, _, err = info.Device.AllocateMemory(nil, core1_0.MemoryAllocateInfo{
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memoryTypeIndex,
	})
	if err != nil {
		log.Fatalln(err)
	}

	/* Map the buffer memory and copy both matrices */
	pData, _, err := info.UniformData.Mem.Map(0, memReqs.Size, 0)
	if err != nil {
		log.Fatalln(err)
	}

	dataBuffer := unsafe.Slice((*byte)(pData), memReqs.Size)

	buf := &bytes.Buffer{}
	err = binary.Write(buf, common.ByteOrder, info.MVP)
	if err != nil {
		log.Fatalln(err)
	}
	err = binary.Write(buf, common.ByteOrder, mvp2)
	if err != nil {
		log.Fatalln(err)
	}

	copy(dataBuffer, buf.Bytes())

	info.UniformData.Mem.Unmap()

	_, err = info.UniformData.Buf.BindBufferMemory(info.UniformData.Mem, 0)
	if err != nil {
		log.Fatalln(err)
	}

	info.UniformData.BufferInfo.Buffer = info.UniformData.Buf
	info.UniformData.BufferInfo.Offset = 0
	info.UniformData.BufferInfo.Range = bufSize

	/* Init desciptor and pipeline layouts - descriptor type is
	 * UNIFORM_BUFFER_DYNAMIC */
	layoutBindings := []core1_0.DescriptorSetLayoutBinding{
		{
			Binding:         0,
			DescriptorType:  core1_0.DescriptorTypeUniformBufferDynamic,
			DescriptorCount: 1,
			StageFlags:      core1_0.StageVertex,
		},
	}

	/* Next take layout bindings and use them to create a descriptor set layout
	 */
	descLayout, _, err := info.Device.CreateDescriptorSetLayout(nil, core1_0.DescriptorSetLayoutCreateInfo{
		Bindings: layoutBindings,
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

	info.DescPool, _, err = info.Device.CreateDescriptorPool(nil, core1_0.DescriptorPoolCreateInfo{
		MaxSets: 1,
		PoolSizes: []core1_0.DescriptorPoolSize{
			{
				Type:            core1_0.DescriptorTypeUniformBufferDynamic,
				DescriptorCount: 1,
			},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	info.DescSet, _, err = info.Device.AllocateDescriptorSets(core1_0.DescriptorSetAllocateInfo{
		DescriptorPool: info.DescPool,
		SetLayouts:     info.DescLayout,
	})
	if err != nil {
		log.Fatalln(err)
	}

	err = info.Device.UpdateDescriptorSets([]core1_0.WriteDescriptorSet{
		{
			DstSet:          info.DescSet[0],
			DstBinding:      0,
			DstArrayElement: 0,
			DescriptorType:  core1_0.DescriptorTypeUniformBufferDynamic,

			BufferInfo: []core1_0.DescriptorBufferInfo{info.UniformData.BufferInfo},
		},
	}, nil)
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

	clearValues := []core1_0.ClearValue{
		core1_0.ClearValueFloat{0.2, 0.2, 0.2, 0.2},
		core1_0.ClearValueDepthStencil{Depth: 1, Stencil: 0},
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

	err = info.Cmd.CmdBeginRenderPass(core1_0.SubpassContentsInline, core1_0.RenderPassBeginInfo{
		RenderPass:  info.RenderPass,
		Framebuffer: info.Framebuffer[info.CurrentBuffer],
		RenderArea: core1_0.Rect2D{
			Offset: core1_0.Offset2D{0, 0},
			Extent: core1_0.Extent2D{info.Width, info.Height},
		},
		ClearValues: clearValues,
	})
	if err != nil {
		log.Fatalln(err)
	}

	info.Cmd.CmdBindPipeline(core1_0.PipelineBindPointGraphics, info.Pipeline)

	/* The first draw should use the first matrix in the buffer */
	info.Cmd.CmdBindDescriptorSets(core1_0.PipelineBindPointGraphics, info.PipelineLayout, info.DescSet, []int{0})

	info.Cmd.CmdBindVertexBuffers([]core1_0.Buffer{info.VertexBuffer.Buf}, []int{0})

	info.InitViewports()
	info.InitScissors()

	info.Cmd.CmdDraw(36, 1, 0, 0)

	/* The second draw should use the
	   second matrix in the buffer */
	info.Cmd.CmdBindDescriptorSets(core1_0.PipelineBindPointGraphics, info.PipelineLayout, info.DescSet, []int{bufSize})
	info.Cmd.CmdDraw(36, 1, 0, 0)

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
			WaitSemaphores:   []core1_0.Semaphore{imageAcquiredSemaphore},
			CommandBuffers:   []core1_0.CommandBuffer{info.Cmd},
			WaitDstStageMask: []core1_0.PipelineStageFlags{core1_0.PipelineStageColorAttachmentOutput},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	/* Now present the image in the window */
	for {
		res, err := info.Device.WaitForFences(true, utils.FenceTimeout, []core1_0.Fence{drawFence})
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
	/* VULKAN_KEY_END */
	if info.SaveImages {
		err = info.WritePNG("push_constants")
		if err != nil {
			log.Fatalln(err)
		}
	}

	drawFence.Destroy(nil)
	imageAcquiredSemaphore.Destroy(nil)
	info.DestroyPipeline()
	info.DestroyPipelineCache()
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
	info.Window.Destroy()
}
