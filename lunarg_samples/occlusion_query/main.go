package main

import (
	"bytes"
	"embed"
	"encoding/binary"
	"fmt"
	"github.com/CannibalVox/VKng/core"
	"github.com/CannibalVox/VKng/core/common"
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

func logDebug(msgType ext_debug_utils.MessageType, severity ext_debug_utils.MessageSeverity, data *ext_debug_utils.CallbackData) bool {
	log.Printf("[%s %s] - %s", severity, msgType, data.Message)
	return false
}

/*
VULKAN_SAMPLE_SHORT_DESCRIPTION
Use occlusion query to determine if drawing renders any samples.
This could be used to quickly determine if more expensive rendering should be
done. Use vkCreateQueryPool, vkCmdResetQueryPool, and vkDestroyQueryPool to
manage a pool. Use vkCmdBeginQuery and vkCmdEndQuery to enclose rendering.
Use vkCmdCopyQueryPoolResults or vkGetQueryPoolResults to read query results.
This example does one query with no rendering to give a zero result and a second
query with rendering to give a non-zero result.  Note that exact counts are not
guaranteed unless vkGetPhysicalDeviceFeatures sets occlusionQueryPrecise and the
VK_QUERY_CONTROL_PRECISE_BIT is set for vkCmdBeginQuery.

This example uses vkCmdCopyQueryPoolResults followed by vkMapMemory of a buffer.
vkCmdCopyQueryPoolResults could also be used to set uniforms used later by
shaders.
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
	debugOptions := &ext_debug_utils.CreationOptions{
		CaptureSeverities: ext_debug_utils.SeverityWarning | ext_debug_utils.SeverityError,
		CaptureTypes:      ext_debug_utils.TypeAll,
		Callback:          logDebug,
	}

	err = info.InitInstance("Occlusion Query", debugOptions)
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

	err = info.InitSwapchain(common.ImageUsageColorAttachment | common.ImageUsageTransferSrc)
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

	err = info.InitDescriptorAndPipelineLayouts(false)
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitRenderPass(true, true, common.LayoutPresentSrcKHR, common.LayoutUndefined)
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

	err = info.InitDescriptorPool(false)
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitDescriptorSet(false)
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

	/* VULKAN_KEY_START */

	clearValues := []core.ClearValue{
		core.ClearValueFloat{0.2, 0.2, 0.2, 0.2},
		core.ClearValueDepthStencil{Depth: 1, Stencil: 0},
	}

	imageAcquiredSemaphore, _, err := info.Loader.CreateSemaphore(info.Device, &core.SemaphoreOptions{})
	if err != nil {
		log.Fatalln(err)
	}

	// Get the index of the next available swapchain image:
	info.CurrentBuffer, _, err = info.Swapchain.AcquireNextImage(common.NoTimeout, imageAcquiredSemaphore, nil)
	if err != nil {
		log.Fatalln(err)
	}

	/* Allocate a uniform buffer that will take query results. */
	queryResultBuf, _, err := info.Loader.CreateBuffer(info.Device, &core.BufferOptions{
		BufferSize:  4 * int(unsafe.Sizeof(uint64(0))),
		Usage:       common.UsageUniformBuffer | common.UsageTransferDst,
		SharingMode: common.SharingExclusive,
	})
	if err != nil {
		log.Fatalln(err)
	}

	memReqs := queryResultBuf.MemoryRequirements()

	memoryTypeIndex, err := info.MemoryTypeFromProperties(memReqs.MemoryType, core.MemoryHostVisible|core.MemoryHostCoherent)
	if err != nil {
		log.Fatalln(err)
	}

	queryResultMem, _, err := info.Device.AllocateMemory(&core.DeviceMemoryOptions{
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memoryTypeIndex,
	})
	if err != nil {
		log.Fatalln(err)
	}

	_, err = queryResultBuf.BindBufferMemory(queryResultMem, 0)
	if err != nil {
		log.Fatalln(err)
	}

	queryPool, _, err := info.Loader.CreateQueryPool(info.Device, &core.QueryPoolOptions{
		QueryType:  common.QueryTypeOcclusion,
		QueryCount: 2,
	})
	if err != nil {
		log.Fatalln(err)
	}

	info.Cmd.CmdResetQueryPool(queryPool, 0, 2)

	err = info.Cmd.CmdBeginRenderPass(core.ContentsInline, &core.RenderPassBeginOptions{
		RenderPass:  info.RenderPass,
		Framebuffer: info.Framebuffer[info.CurrentBuffer],
		RenderArea: common.Rect2D{
			Offset: common.Offset2D{0, 0},
			Extent: common.Extent2D{Width: info.Width, Height: info.Height},
		},
		ClearValues: clearValues,
	})
	if err != nil {
		log.Fatalln(err)
	}

	info.Cmd.CmdBindPipeline(common.BindGraphics, info.Pipeline)
	info.Cmd.CmdBindDescriptorSets(common.BindGraphics, info.PipelineLayout, info.DescSet, nil)

	info.Cmd.CmdBindVertexBuffers([]core.Buffer{info.VertexBuffer.Buf}, []int{0})

	info.Cmd.CmdSetViewport([]common.Viewport{
		{
			X: 0, Y: 0,
			MinDepth: 0, MaxDepth: 1,
			Width:  float32(info.Width),
			Height: float32(info.Height),
		},
	})

	info.Cmd.CmdSetScissor([]common.Rect2D{
		{
			Offset: common.Offset2D{0, 0},
			Extent: common.Extent2D{info.Width, info.Height},
		},
	})

	info.Cmd.CmdBeginQuery(queryPool, 0, 0)
	info.Cmd.CmdEndQuery(queryPool, 0)

	info.Cmd.CmdBeginQuery(queryPool, 1, 0)

	info.Cmd.CmdDraw(36, 1, 0, 0)
	info.Cmd.CmdEndRenderPass()

	info.Cmd.CmdEndQuery(queryPool, 1)
	info.Cmd.CmdCopyQueryPoolResults(queryPool, 0, 2, queryResultBuf, 0, int(unsafe.Sizeof(uint64(0))), common.QueryResult64Bit|common.QueryResultWait)

	_, err = info.Cmd.End()
	if err != nil {
		log.Fatalln(err)
	}

	drawFence, _, err := info.Loader.CreateFence(info.Device, &core.FenceOptions{})
	if err != nil {
		log.Fatalln(err)
	}

	/* Queue the command buffer for execution */
	_, err = info.GraphicsQueue.SubmitToQueue(drawFence, []*core.SubmitOptions{
		{
			WaitSemaphores: []core.Semaphore{imageAcquiredSemaphore},
			WaitDstStages:  []common.PipelineStages{common.PipelineStageColorAttachmentOutput},
			CommandBuffers: []core.CommandBuffer{info.Cmd},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	_, err = info.GraphicsQueue.WaitForIdle()
	if err != nil {
		log.Fatalln(err)
	}

	resultData, _, err := queryPool.PopulateResults(0, 2, 32, 8, common.QueryResult64Bit|common.QueryResultWait)
	if err != nil {
		log.Fatalln(err)
	}

	resultReader := bytes.NewBuffer(resultData)
	samplesPassed := []uint64{0, 0, 0, 0}
	err = binary.Read(resultReader, common.ByteOrder, samplesPassed)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("vkGetQueryPoolResults data")
	fmt.Printf("samplesPassed[0] = %d\n", samplesPassed[0])
	fmt.Printf("samplesPassed[1] = %d\n", samplesPassed[1])

	/* Read back query result from buffer */
	samplesPassedPtr, _, err := queryResultMem.MapMemory(0, memReqs.Size, 0)
	if err != nil {
		log.Fatalln(err)
	}
	samplesPassedBuffer := ([]uint64)(unsafe.Slice((*uint64)(samplesPassedPtr), 4))

	fmt.Println("vkCmdCopyQueryPoolResults  data")
	fmt.Printf("samplesPassed[0] = %d\n", samplesPassedBuffer[0])
	fmt.Printf("samplesPassed[1] = %d\n", samplesPassedBuffer[1])

	queryResultMem.UnmapMemory()

	/* Now present the image in the window */

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

	_, err = info.PresentQueue.WaitForIdle()
	if err != nil {
		log.Fatalln(err)
	}

	_, _, err = info.Swapchain.PresentToQueue(info.PresentQueue, &khr_swapchain.PresentOptions{
		Swapchains:   []khr_swapchain.Swapchain{info.Swapchain},
		ImageIndices: []int{info.CurrentBuffer},
	})
	if err != nil {
		log.Fatalln(err)
	}

	time.Sleep(time.Second)

	/* VULKAN_KEY_END */
	if info.SaveImages {
		err = info.WritePNG("occlusion_query")
		if err != nil {
			log.Fatalln(err)
		}
	}

	queryResultBuf.Destroy()
	info.Device.FreeMemory(queryResultMem)
	imageAcquiredSemaphore.Destroy()
	queryPool.Destroy()
	drawFence.Destroy()
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

	info.Surface.Destroy()
	debugMessenger.Destroy()
	info.DestroyInstance()
	err = info.Window.Destroy()
	if err != nil {
		log.Fatalln(err)
	}
}