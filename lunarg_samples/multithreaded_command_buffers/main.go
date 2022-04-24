package main

import (
	"context"
	"embed"
	"github.com/CannibalVox/VKng/core"
	"github.com/CannibalVox/VKng/core/common"
	"github.com/CannibalVox/VKng/core/core1_0"
	"github.com/CannibalVox/VKng/examples/lunarg_samples/utils"
	"github.com/CannibalVox/VKng/extensions/ext_debug_utils"
	"github.com/CannibalVox/VKng/extensions/khr_swapchain"
	"github.com/veandco/go-sdl2/sdl"
	"golang.org/x/sync/errgroup"
	"log"
	"runtime/debug"
	"time"
	"unsafe"
)

//go:embed shaders
var fileSystem embed.FS

func logDebug(msgType ext_debug_utils.MessageTypes, severity ext_debug_utils.MessageSeverities, data *ext_debug_utils.CallbackDataOptions) bool {
	log.Printf("[%s %s] - %s", severity, msgType, data.Message)
	debug.PrintStack()
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
	debugOptions := ext_debug_utils.CreateOptions{
		CaptureSeverities: ext_debug_utils.SeverityWarning | ext_debug_utils.SeverityError,
		CaptureTypes:      ext_debug_utils.TypeGeneral | ext_debug_utils.TypeValidation | ext_debug_utils.TypePerformance,
		Callback:          logDebug,
	}

	err = info.InitInstance("MT Cmd Buffer Sample", debugOptions)
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

	err = info.InitSwapchain(core1_0.ImageUsageColorAttachment | core1_0.ImageUsageTransferDst)
	if err != nil {
		log.Fatalln(err)
	}

	info.ImageAcquiredSemaphore, _, err = info.Loader.CreateSemaphore(info.Device, nil, core1_0.SemaphoreOptions{})
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

	err = info.SetImageLayout(info.Buffers[info.CurrentBuffer].Image, core1_0.AspectColor, core1_0.ImageLayoutUndefined, core1_0.ImageLayoutColorAttachmentOptimal, core1_0.PipelineStageTopOfPipe, core1_0.PipelineStageColorAttachmentOutput)
	if err != nil {
		log.Fatalln(err)
	}

	info.PipelineLayout, _, err = info.Loader.CreatePipelineLayout(info.Device, nil, core1_0.PipelineLayoutOptions{})
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
	info.VertexBinding = core1_0.VertexBindingDescription{
		Binding:   0,
		InputRate: core1_0.RateVertex,
		Stride:    int(unsafe.Sizeof(triData[0])),
	}

	info.VertexAttributes = []core1_0.VertexAttributeDescription{
		{
			Binding:  0,
			Location: 0,
			Format:   core1_0.DataFormatR32G32B32A32SignedFloat,
			Offset:   0,
		},
		{
			Binding:  0,
			Location: 1,
			Format:   core1_0.DataFormatR32G32B32A32SignedFloat,
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

	srRange := common.ImageSubresourceRange{
		AspectMask:     core1_0.AspectColor,
		BaseMipLevel:   0,
		LevelCount:     1,
		BaseArrayLayer: 0,
		LayerCount:     1,
	}

	clearColor := common.ClearValueFloat{0.2, 0.2, 0.2, 0.2}

	/* We need to do the clear here instead of as a load op since all 3 threads
	 * share the same pipeline / renderpass */
	err = info.SetImageLayout(info.Buffers[info.CurrentBuffer].Image, core1_0.AspectColor, core1_0.ImageLayoutColorAttachmentOptimal, core1_0.ImageLayoutTransferDstOptimal, core1_0.PipelineStageColorAttachmentOutput, core1_0.PipelineStageTransfer)
	if err != nil {
		log.Fatalln(err)
	}
	info.Cmd.CmdClearColorImage(info.Buffers[info.CurrentBuffer].Image, core1_0.ImageLayoutTransferDstOptimal, clearColor, []common.ImageSubresourceRange{srRange})
	err = info.SetImageLayout(info.Buffers[info.CurrentBuffer].Image, core1_0.AspectColor, core1_0.ImageLayoutTransferDstOptimal, core1_0.ImageLayoutColorAttachmentOptimal, core1_0.PipelineStageTransfer, core1_0.PipelineStageColorAttachmentOutput)
	if err != nil {
		log.Fatalln(err)
	}

	_, err = info.Cmd.End()
	if err != nil {
		log.Fatalln(err)
	}

	clearFence, err := info.InitFence()
	if err != nil {
		log.Fatalln(err)
	}

	/* Queue the command buffer for execution */
	_, err = info.GraphicsQueue.SubmitToQueue(clearFence, []core1_0.SubmitOptions{
		{
			WaitSemaphores: []core1_0.Semaphore{info.ImageAcquiredSemaphore},
			WaitDstStages:  []common.PipelineStages{core1_0.PipelineStageColorAttachmentOutput},
			CommandBuffers: []core1_0.CommandBuffer{info.Cmd},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	for {
		res, err := clearFence.Wait(utils.FenceTimeout)
		if err != nil {
			log.Fatalln(err)
		}

		if res != core1_0.VKTimeout {
			break
		}
	}
	clearFence.Destroy(nil)

	/* VULKAN_KEY_START */
	group, _ := errgroup.WithContext(context.Background())
	for i := 0; i < 3; i++ {
		idx := i
		group.Go(func() error {
			return perThreadCode(info, idx)
		})
	}

	_, err = info.Cmd.Begin(core1_0.BeginOptions{})
	if err != nil {
		log.Fatalln(err)
	}

	err = info.Cmd.CmdPipelineBarrier(core1_0.PipelineStageColorAttachmentOutput,
		core1_0.PipelineStageBottomOfPipe,
		0,
		nil,
		nil,
		[]core1_0.ImageMemoryBarrierOptions{
			{
				SrcAccessMask:       core1_0.AccessColorAttachmentWrite,
				DstAccessMask:       core1_0.AccessMemoryRead,
				OldLayout:           core1_0.ImageLayoutColorAttachmentOptimal,
				NewLayout:           khr_swapchain.ImageLayoutPresentSrc,
				SrcQueueFamilyIndex: -1,
				DstQueueFamilyIndex: -1,
				SubresourceRange: common.ImageSubresourceRange{
					AspectMask:     core1_0.AspectColor,
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

	_, err = info.Cmd.End()
	if err != nil {
		log.Fatalln(err)
	}

	/* Wait for all of the threads to finish */
	err = group.Wait()
	if err != nil {
		log.Fatalln(err)
	}

	drawFence, _, err := info.Loader.CreateFence(info.Device, nil, core1_0.FenceOptions{})
	if err != nil {
		log.Fatalln(err)
	}

	/* Queue the command buffer for execution */
	_, err = info.GraphicsQueue.SubmitToQueue(drawFence, []core1_0.SubmitOptions{
		{
			CommandBuffers: []core1_0.CommandBuffer{commandBuffers[0], commandBuffers[1], commandBuffers[2], info.Cmd},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

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

	err = info.ExecutePresentImage()
	if err != nil {
		log.Fatalln(err)
	}

	time.Sleep(time.Second)

	/* VULKAN_KEY_END */
	if info.SaveImages {
		err = info.WritePNG("multithreaded_command_buffers")
		if err != nil {
			log.Fatalln(err)
		}
	}

	for i := 0; i < 3; i++ {
		vertexBuffers[i].Buffer.Destroy(nil)
		vertexBuffers[i].Mem.Free(nil)
		info.Loader.FreeCommandBuffers([]core1_0.CommandBuffer{commandBuffers[i]})
		commandPools[i].Destroy(nil)
	}
	info.ImageAcquiredSemaphore.Destroy(nil)
	drawFence.Destroy(nil)
	info.DestroyPipeline()
	info.DestroyPipelineCache()
	info.DestroyFramebuffers()
	info.DestroyShaders()
	info.DestroyRenderpass()
	info.PipelineLayout.Destroy(nil)
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

func perThreadCode(info *utils.SampleInfo, i int) error {
	/* This code should be executed by each of the three threads.  It will  */
	/* create a vertex buffer with position and color per vertex, then load */
	/* commands into the thread's designated command buffer to draw the     */
	/* triangle                                                             */
	var err error

	commandPools[i], _, err = info.Loader.CreateCommandPool(info.Device, nil, core1_0.CommandPoolOptions{
		GraphicsQueueFamily: &info.GraphicsQueueFamilyIndex,
	})
	if err != nil {
		return err
	}

	buffers, _, err := info.Loader.AllocateCommandBuffers(core1_0.CommandBufferOptions{
		CommandPool: commandPools[i],
		Level:       core1_0.LevelPrimary,
		BufferCount: 1,
	})
	if err != nil {
		return err
	}
	commandBuffers[i] = buffers[0]

	vertexBuffer, _, err := info.Loader.CreateBuffer(info.Device, nil, core1_0.BufferOptions{
		BufferSize:  3 * int(unsafe.Sizeof(triData[0])),
		Usage:       core1_0.BufferUsageVertexBuffer,
		SharingMode: core1_0.SharingExclusive,
	})
	if err != nil {
		return err
	}

	memReqs := vertexBuffer.MemoryRequirements()

	memoryTypeIndex, err := info.MemoryTypeFromProperties(memReqs.MemoryType, core1_0.MemoryPropertyHostVisible|core1_0.MemoryPropertyHostCoherent)
	if err != nil {
		return err
	}

	vertexMem, _, err := info.Loader.AllocateMemory(info.Device, nil, core1_0.DeviceMemoryOptions{
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

	data, _, err := vertexMem.MapMemory(0, memReqs.Size, 0)
	if err != nil {
		return err
	}

	vertexPtr := (*Vertex)(data)
	vertexSlice := ([]Vertex)(unsafe.Slice(vertexPtr, 3))
	copy(vertexSlice, triData[i*3:i*3+3])

	vertexMem.UnmapMemory()

	_, err = vertexBuffer.BindBufferMemory(vertexMem, 0)
	if err != nil {
		return err
	}

	_, err = buffers[0].Begin(core1_0.BeginOptions{})
	if err != nil {
		return err
	}

	err = buffers[0].CmdBeginRenderPass(core1_0.SubpassContentsInline, core1_0.RenderPassBeginOptions{
		RenderPass:  info.RenderPass,
		Framebuffer: info.Framebuffer[info.CurrentBuffer],
		RenderArea: common.Rect2D{
			Offset: common.Offset2D{0, 0},
			Extent: common.Extent2D{info.Width, info.Height},
		},
	})
	if err != nil {
		return err
	}

	buffers[0].CmdBindPipeline(core1_0.BindGraphics, info.Pipeline)
	buffers[0].CmdBindVertexBuffers([]core1_0.Buffer{vertexBuffer}, []int{0})
	buffers[0].CmdSetViewport([]common.Viewport{
		{
			X: 0, Y: 0,
			MinDepth: 0, MaxDepth: 1,
			Width:  float32(info.Width),
			Height: float32(info.Height),
		},
	})
	buffers[0].CmdSetScissor([]common.Rect2D{
		{
			Offset: common.Offset2D{0, 0},
			Extent: common.Extent2D{info.Width, info.Height},
		},
	})

	buffers[0].CmdDraw(3, 1, 0, 0)
	buffers[0].CmdEndRenderPass()

	_, err = buffers[0].End()
	return err
}
