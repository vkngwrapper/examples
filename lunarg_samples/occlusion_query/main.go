package main

import (
	"bytes"
	"embed"
	"encoding/binary"
	"fmt"
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

func logDebug(msgType ext_debug_utils.MessageTypes, severity ext_debug_utils.MessageSeverities, data *ext_debug_utils.DebugUtilsMessengerCallbackData) bool {
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
	debugOptions := ext_debug_utils.DebugUtilsMessengerCreateInfo{
		MessageSeverity: ext_debug_utils.SeverityWarning | ext_debug_utils.SeverityError,
		MessageType:     ext_debug_utils.TypeGeneral | ext_debug_utils.TypeValidation | ext_debug_utils.TypePerformance,
		UserCallback:    logDebug,
	}

	err = info.InitInstance("Occlusion Query", debugOptions)
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

	err = info.InitDescriptorAndPipelineLayouts(false)
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
	if err != nil {
		log.Fatalln(err)
	}

	/* Allocate a uniform buffer that will take query results. */
	queryResultBuf, _, err := info.Device.CreateBuffer(nil, core1_0.BufferCreateInfo{
		Size:        4 * int(unsafe.Sizeof(uint64(0))),
		Usage:       core1_0.BufferUsageUniformBuffer | core1_0.BufferUsageTransferDst,
		SharingMode: core1_0.SharingModeExclusive,
	})
	if err != nil {
		log.Fatalln(err)
	}

	memReqs := queryResultBuf.MemoryRequirements()

	memoryTypeIndex, err := info.MemoryTypeFromProperties(memReqs.MemoryTypeBits, core1_0.MemoryPropertyHostVisible|core1_0.MemoryPropertyHostCoherent)
	if err != nil {
		log.Fatalln(err)
	}

	queryResultMem, _, err := info.Device.AllocateMemory(nil, core1_0.MemoryAllocateInfo{
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

	queryPool, _, err := info.Device.CreateQueryPool(nil, core1_0.QueryPoolCreateInfo{
		QueryType:  core1_0.QueryTypeOcclusion,
		QueryCount: 2,
	})
	if err != nil {
		log.Fatalln(err)
	}

	info.Cmd.CmdResetQueryPool(queryPool, 0, 2)

	err = info.Cmd.CmdBeginRenderPass(core1_0.SubpassContentsInline, core1_0.RenderPassBeginInfo{
		RenderPass:  info.RenderPass,
		Framebuffer: info.Framebuffer[info.CurrentBuffer],
		RenderArea: core1_0.Rect2D{
			Offset: core1_0.Offset2D{0, 0},
			Extent: core1_0.Extent2D{Width: info.Width, Height: info.Height},
		},
		ClearValues: clearValues,
	})
	if err != nil {
		log.Fatalln(err)
	}

	info.Cmd.CmdBindPipeline(core1_0.PipelineBindPointGraphics, info.Pipeline)
	info.Cmd.CmdBindDescriptorSets(core1_0.PipelineBindPointGraphics, info.PipelineLayout, info.DescSet, nil)

	info.Cmd.CmdBindVertexBuffers([]core1_0.Buffer{info.VertexBuffer.Buf}, []int{0})

	info.Cmd.CmdSetViewport([]core1_0.Viewport{
		{
			X: 0, Y: 0,
			MinDepth: 0, MaxDepth: 1,
			Width:  float32(info.Width),
			Height: float32(info.Height),
		},
	})

	info.Cmd.CmdSetScissor([]core1_0.Rect2D{
		{
			Offset: core1_0.Offset2D{0, 0},
			Extent: core1_0.Extent2D{info.Width, info.Height},
		},
	})

	info.Cmd.CmdBeginQuery(queryPool, 0, 0)
	info.Cmd.CmdEndQuery(queryPool, 0)

	info.Cmd.CmdBeginQuery(queryPool, 1, 0)

	info.Cmd.CmdDraw(36, 1, 0, 0)
	info.Cmd.CmdEndRenderPass()

	info.Cmd.CmdEndQuery(queryPool, 1)
	info.Cmd.CmdCopyQueryPoolResults(queryPool, 0, 2, queryResultBuf, 0, int(unsafe.Sizeof(uint64(0))), core1_0.QueryResult64Bit|core1_0.QueryResultWait)

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
			WaitDstStageMask: []core1_0.PipelineStageFlags{core1_0.PipelineStageColorAttachmentOutput},
			CommandBuffers:   []core1_0.CommandBuffer{info.Cmd},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	_, err = info.GraphicsQueue.WaitIdle()
	if err != nil {
		log.Fatalln(err)
	}

	resultsData := make([]byte, 32)
	_, err = queryPool.PopulateResults(0, 2, resultsData, 8, core1_0.QueryResult64Bit|core1_0.QueryResultWait)
	if err != nil {
		log.Fatalln(err)
	}

	resultReader := bytes.NewBuffer(resultsData)
	samplesPassed := []uint64{0, 0, 0, 0}
	err = binary.Read(resultReader, common.ByteOrder, samplesPassed)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("vkGetQueryPoolResults data")
	fmt.Printf("samplesPassed[0] = %d\n", samplesPassed[0])
	fmt.Printf("samplesPassed[1] = %d\n", samplesPassed[1])

	/* Read back query result from buffer */
	samplesPassedPtr, _, err := queryResultMem.Map(0, memReqs.Size, 0)
	if err != nil {
		log.Fatalln(err)
	}
	samplesPassedBuffer := ([]uint64)(unsafe.Slice((*uint64)(samplesPassedPtr), 4))

	fmt.Println("vkCmdCopyQueryPoolResults  data")
	fmt.Printf("samplesPassed[0] = %d\n", samplesPassedBuffer[0])
	fmt.Printf("samplesPassed[1] = %d\n", samplesPassedBuffer[1])

	queryResultMem.Unmap()

	/* Now present the image in the window */

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

	_, err = info.PresentQueue.WaitIdle()
	if err != nil {
		log.Fatalln(err)
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
		err = info.WritePNG("occlusion_query")
		if err != nil {
			log.Fatalln(err)
		}
	}

	queryResultBuf.Destroy(nil)
	queryResultMem.Free(nil)
	imageAcquiredSemaphore.Destroy(nil)
	queryPool.Destroy(nil)
	drawFence.Destroy(nil)
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
	err = info.Window.Destroy()
	if err != nil {
		log.Fatalln(err)
	}
}
