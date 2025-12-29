package main

import (
	"context"
	"embed"
	"github.com/loov/hrtime"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/vkngwrapper/core/v3"
	"github.com/vkngwrapper/core/v3/common"
	"github.com/vkngwrapper/core/v3/core1_0"
	"github.com/vkngwrapper/examples/lunarg_samples/utils"
	"github.com/vkngwrapper/extensions/v3/ext_debug_utils"
	"github.com/vkngwrapper/extensions/v3/khr_swapchain"
	"golang.org/x/sync/errgroup"
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

type Vertex struct {
	PosX, PosY, PosZ, PosW float32 // Position Data
	R, G, B, A             float32 // Color
}

var triData = []Vertex{
	{-0.25, -0.25, 0, 1, 1, 0, 0, 1},
	{0.25, -0.25, 0, 1, 1, 0, 0, 1},
	{0, 0.25, 0, 1, 1, 0, 0, 1},
	{-0.75, -0.25, 0, 1, 0, 1, 0, 1},
	{-0.25, -0.25, 0, 1, 0, 1, 0, 1},
	{-0.5, 0.25, 0, 1, 0, 1, 0, 1},
	{0.25, -0.25, 0, 1, 0, 0, 1, 1},
	{0.75, -0.25, 0, 1, 0, 0, 1, 1},
	{0.5, 0.25, 0, 1, 0, 0, 1, 1},
}

type vertexData struct {
	Buffer core1_0.Buffer
	Mem    core1_0.DeviceMemory
}

var commandPools [3]core1_0.CommandPool
var commandBuffers [3]core1_0.CommandBuffer
var vertexBuffers [3]vertexData

/*
VULKAN_SAMPLE_SHORT_DESCRIPTION
Use per-thread command buffers to draw 3 triangles
*/

/* Set up Vulkan pipeline and use three threads to create 3       */
/* command buffers, each using a vertex buffer to draw a triangle */
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

	err = info.InitInstance("MT Cmd Buffer Sample", debugOptions)
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

	err = info.InitSwapchain(core1_0.ImageUsageColorAttachment | core1_0.ImageUsageTransferDst)
	if err != nil {
		log.Fatalln(err)
	}

	info.ImageAcquiredSemaphore, _, err = info.DeviceDriver.CreateSemaphore(nil, core1_0.SemaphoreCreateInfo{})
	if err != nil {
		log.Fatalln(err)
	}

	// Get the index of the next available swapchain image:
	info.CurrentBuffer, _, err = info.SwapchainExtension.AcquireNextImage(info.Swapchain, common.NoTimeout, &info.ImageAcquiredSemaphore, nil)
	// TODO: Deal with the VK_SUBOPTIMAL_KHR and VK_ERROR_OUT_OF_DATE_KHR
	// return codes
	if err != nil {
		log.Fatalln(err)
	}

	err = info.SetImageLayout(info.Buffers[info.CurrentBuffer].Image, core1_0.ImageAspectColor, core1_0.ImageLayoutUndefined, core1_0.ImageLayoutColorAttachmentOptimal, core1_0.PipelineStageTopOfPipe, core1_0.PipelineStageColorAttachmentOutput)
	if err != nil {
		log.Fatalln(err)
	}

	info.PipelineLayout, _, err = info.DeviceDriver.CreatePipelineLayout(nil, core1_0.PipelineLayoutCreateInfo{})
	if err != nil {
		log.Fatalln(err)
	}

	// Can't clear in renderpass load because we re-use pipeline
	err = info.InitRenderPass(false, false, core1_0.ImageLayoutColorAttachmentOptimal, core1_0.ImageLayoutColorAttachmentOptimal)
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

	/* The binding and attributes should be the same for all 3 vertex buffers,
	 * so init here */
	info.VertexBinding = core1_0.VertexInputBindingDescription{
		Binding:   0,
		InputRate: core1_0.VertexInputRateVertex,
		Stride:    int(unsafe.Sizeof(triData[0])),
	}

	info.VertexAttributes = []core1_0.VertexInputAttributeDescription{
		{
			Binding:  0,
			Location: 0,
			Format:   core1_0.FormatR32G32B32A32SignedFloat,
			Offset:   0,
		},
		{
			Binding:  0,
			Location: 1,
			Format:   core1_0.FormatR32G32B32A32SignedFloat,
			Offset:   int(unsafe.Offsetof(triData[0].R)),
		},
	}

	err = info.InitPipelineCache()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitPipeline(false, true)
	if err != nil {
		log.Fatalln(err)
	}

	srRange := core1_0.ImageSubresourceRange{
		AspectMask:     core1_0.ImageAspectColor,
		BaseMipLevel:   0,
		LevelCount:     1,
		BaseArrayLayer: 0,
		LayerCount:     1,
	}

	clearColor := core1_0.ClearValueFloat{0.2, 0.2, 0.2, 0.2}

	/* We need to do the clear here instead of as a load op since all 3 threads
	 * share the same pipeline / renderpass */
	err = info.SetImageLayout(info.Buffers[info.CurrentBuffer].Image, core1_0.ImageAspectColor, core1_0.ImageLayoutColorAttachmentOptimal, core1_0.ImageLayoutTransferDstOptimal, core1_0.PipelineStageColorAttachmentOutput, core1_0.PipelineStageTransfer)
	if err != nil {
		log.Fatalln(err)
	}
	info.DeviceDriver.CmdClearColorImage(info.Cmd, info.Buffers[info.CurrentBuffer].Image, core1_0.ImageLayoutTransferDstOptimal, clearColor, srRange)
	err = info.SetImageLayout(info.Buffers[info.CurrentBuffer].Image, core1_0.ImageAspectColor, core1_0.ImageLayoutTransferDstOptimal, core1_0.ImageLayoutColorAttachmentOptimal, core1_0.PipelineStageTransfer, core1_0.PipelineStageColorAttachmentOutput)
	if err != nil {
		log.Fatalln(err)
	}

	_, err = info.DeviceDriver.EndCommandBuffer(info.Cmd)
	if err != nil {
		log.Fatalln(err)
	}

	clearFence, err := info.InitFence()
	if err != nil {
		log.Fatalln(err)
	}

	/* Queue the command buffer for execution */
	_, err = info.DeviceDriver.QueueSubmit(info.GraphicsQueue, &clearFence,
		core1_0.SubmitInfo{
			WaitSemaphores:   []core1_0.Semaphore{info.ImageAcquiredSemaphore},
			WaitDstStageMask: []core1_0.PipelineStageFlags{core1_0.PipelineStageColorAttachmentOutput},
			CommandBuffers:   []core1_0.CommandBuffer{info.Cmd},
		},
	)
	if err != nil {
		log.Fatalln(err)
	}

	for {
		res, err := info.DeviceDriver.WaitForFences(true, utils.FenceTimeout, clearFence)
		if err != nil {
			log.Fatalln(err)
		}

		if res != core1_0.VKTimeout {
			break
		}
	}
	info.DeviceDriver.DestroyFence(clearFence, nil)

	/* VULKAN_KEY_START */
	group, _ := errgroup.WithContext(context.Background())
	for i := 0; i < 3; i++ {
		idx := i
		group.Go(func() error {
			return perThreadCode(info, idx)
		})
	}

	_, err = info.DeviceDriver.BeginCommandBuffer(info.Cmd, core1_0.CommandBufferBeginInfo{})
	if err != nil {
		log.Fatalln(err)
	}

	err = info.DeviceDriver.CmdPipelineBarrier(info.Cmd, core1_0.PipelineStageColorAttachmentOutput,
		core1_0.PipelineStageBottomOfPipe,
		0,
		nil,
		nil,
		[]core1_0.ImageMemoryBarrier{
			{
				SrcAccessMask:       core1_0.AccessColorAttachmentWrite,
				DstAccessMask:       core1_0.AccessMemoryRead,
				OldLayout:           core1_0.ImageLayoutColorAttachmentOptimal,
				NewLayout:           khr_swapchain.ImageLayoutPresentSrc,
				SrcQueueFamilyIndex: -1,
				DstQueueFamilyIndex: -1,
				SubresourceRange: core1_0.ImageSubresourceRange{
					AspectMask:     core1_0.ImageAspectColor,
					BaseMipLevel:   0,
					LevelCount:     1,
					BaseArrayLayer: 0,
					LayerCount:     1,
				},
				Image: info.Buffers[info.CurrentBuffer].Image,
			},
		})
	if err != nil {
		log.Fatalln(err)
	}

	_, err = info.DeviceDriver.EndCommandBuffer(info.Cmd)
	if err != nil {
		log.Fatalln(err)
	}

	/* Wait for all of the threads to finish */
	err = group.Wait()
	if err != nil {
		log.Fatalln(err)
	}

	drawFence, _, err := info.DeviceDriver.CreateFence(nil, core1_0.FenceCreateInfo{})
	if err != nil {
		log.Fatalln(err)
	}

	/* Queue the command buffer for execution */
	_, err = info.DeviceDriver.QueueSubmit(info.GraphicsQueue, &drawFence,
		core1_0.SubmitInfo{
			CommandBuffers: []core1_0.CommandBuffer{commandBuffers[0], commandBuffers[1], commandBuffers[2], info.Cmd},
		},
	)
	if err != nil {
		log.Fatalln(err)
	}

	/* Make sure command buffer is finished before presenting */
	for {
		res, err := info.DeviceDriver.WaitForFences(true, utils.FenceTimeout, drawFence)
		if err != nil {
			log.Fatalln(err)
		}

		if res != core1_0.VKTimeout {
			break
		}
	}

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
		err = info.WritePNG("multithreaded_command_buffers")
		if err != nil {
			log.Fatalln(err)
		}
	}

	for i := 0; i < 3; i++ {
		info.DeviceDriver.DestroyBuffer(vertexBuffers[i].Buffer, nil)
		info.DeviceDriver.FreeMemory(vertexBuffers[i].Mem, nil)
		info.DeviceDriver.FreeCommandBuffers(commandBuffers[i])
		info.DeviceDriver.DestroyCommandPool(commandPools[i], nil)
	}
	info.DeviceDriver.DestroySemaphore(info.ImageAcquiredSemaphore, nil)
	info.DeviceDriver.DestroyFence(drawFence, nil)
	info.DestroyPipeline()
	info.DestroyPipelineCache()
	info.DestroyFramebuffers()
	info.DestroyShaders()
	info.DestroyRenderpass()
	info.DeviceDriver.DestroyPipelineLayout(info.PipelineLayout, nil)
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

func perThreadCode(info *utils.SampleInfo, i int) error {
	/* This code should be executed by each of the three threads.  It will  */
	/* create a vertex buffer with position and color per vertex, then load */
	/* commands into the thread's designated command buffer to draw the     */
	/* triangle                                                             */
	var err error

	commandPools[i], _, err = info.DeviceDriver.CreateCommandPool(nil, core1_0.CommandPoolCreateInfo{
		QueueFamilyIndex: info.GraphicsQueueFamilyIndex,
	})
	if err != nil {
		return err
	}

	buffers, _, err := info.DeviceDriver.AllocateCommandBuffers(core1_0.CommandBufferAllocateInfo{
		CommandPool:        commandPools[i],
		Level:              core1_0.CommandBufferLevelPrimary,
		CommandBufferCount: 1,
	})
	if err != nil {
		return err
	}
	commandBuffers[i] = buffers[0]

	vertexBuffer, _, err := info.DeviceDriver.CreateBuffer(nil, core1_0.BufferCreateInfo{
		Size:        3 * int(unsafe.Sizeof(triData[0])),
		Usage:       core1_0.BufferUsageVertexBuffer,
		SharingMode: core1_0.SharingModeExclusive,
	})
	if err != nil {
		return err
	}

	memReqs := info.DeviceDriver.GetBufferMemoryRequirements(vertexBuffer)

	memoryTypeIndex, err := info.MemoryTypeFromProperties(memReqs.MemoryTypeBits, core1_0.MemoryPropertyHostVisible|core1_0.MemoryPropertyHostCoherent)
	if err != nil {
		return err
	}

	vertexMem, _, err := info.DeviceDriver.AllocateMemory(nil, core1_0.MemoryAllocateInfo{
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memoryTypeIndex,
	})
	if err != nil {
		return err
	}

	vertexBuffers[i] = vertexData{
		Buffer: vertexBuffer,
		Mem:    vertexMem,
	}

	data, _, err := info.DeviceDriver.MapMemory(vertexMem, 0, memReqs.Size, 0)
	if err != nil {
		return err
	}

	vertexPtr := (*Vertex)(data)
	vertexSlice := ([]Vertex)(unsafe.Slice(vertexPtr, 3))
	copy(vertexSlice, triData[i*3:i*3+3])

	info.DeviceDriver.UnmapMemory(vertexMem)

	_, err = info.DeviceDriver.BindBufferMemory(vertexBuffer, vertexMem, 0)
	if err != nil {
		return err
	}

	_, err = info.DeviceDriver.BeginCommandBuffer(buffers[0], core1_0.CommandBufferBeginInfo{})
	if err != nil {
		return err
	}

	err = info.DeviceDriver.CmdBeginRenderPass(buffers[0], core1_0.SubpassContentsInline, core1_0.RenderPassBeginInfo{
		RenderPass:  info.RenderPass,
		Framebuffer: info.Framebuffer[info.CurrentBuffer],
		RenderArea: core1_0.Rect2D{
			Offset: core1_0.Offset2D{0, 0},
			Extent: core1_0.Extent2D{info.Width, info.Height},
		},
	})
	if err != nil {
		return err
	}

	info.DeviceDriver.CmdBindPipeline(buffers[0], core1_0.PipelineBindPointGraphics, info.Pipeline)
	info.DeviceDriver.CmdBindVertexBuffers(buffers[0], 0, []core1_0.Buffer{vertexBuffer}, []int{0})
	info.DeviceDriver.CmdSetViewport(buffers[0],
		core1_0.Viewport{
			X: 0, Y: 0,
			MinDepth: 0, MaxDepth: 1,
			Width:  float32(info.Width),
			Height: float32(info.Height),
		})
	info.DeviceDriver.CmdSetScissor(buffers[0],
		core1_0.Rect2D{
			Offset: core1_0.Offset2D{0, 0},
			Extent: core1_0.Extent2D{info.Width, info.Height},
		})

	info.DeviceDriver.CmdDraw(buffers[0], 3, 1, 0, 0)
	info.DeviceDriver.CmdEndRenderPass(buffers[0])

	_, err = info.DeviceDriver.EndCommandBuffer(buffers[0])
	return err
}
