package main

import (
	"embed"
	"encoding/binary"
	"github.com/CannibalVox/VKng/core"
	"github.com/CannibalVox/VKng/core/common"
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

func logDebug(msgType ext_debug_utils.MessageTypes, severity ext_debug_utils.MessageSeverities, data *ext_debug_utils.CallbackData) bool {
	log.Printf("[%s %s] - %s", severity, msgType, data.Message)
	debug.PrintStack()
	log.Println()
	return false
}

/*
VULKAN_SAMPLE_SHORT_DESCRIPTION
Render two multi-subpass render passes with different framebuffer attachments
*/
/**
 *  Sample using multiple render passes per framebuffer (different x,y extents)
 *  and multiple subpasses per renderpass.
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

	err = info.InitInstance("Multi-pass render passes", debugOptions)
	if err != nil {
		log.Fatalln(err)
	}

	debugLoader := ext_debug_utils.CreateLoaderFromInstance(info.Instance)
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

	err = info.InitSwapchain(common.ImageUsageColorAttachment | common.ImageUsageTransferSrc)
	if err != nil {
		log.Fatalln(err)
	}

	props := info.Gpus[0].FormatProperties(common.FormatD32SignedFloatS8UnsignedInt)
	if (props.LinearTilingFeatures&common.FormatFeatureDepthStencilAttachment != 0) ||
		(props.OptimalTilingFeatures&common.FormatFeatureDepthStencilAttachment != 0) {
		info.Depth.Format = common.FormatD32SignedFloatS8UnsignedInt
	} else {
		info.Depth.Format = common.FormatD24UnsignedNormalizedS8UnsignedInt
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

	/* VULKAN_KEY_START */

	/**
	 *  First renderpass in this sample.
	 *  Stenciled rendering: subpass 1 draw to stencil buffer, subpass 2 draw to
	 *  color buffer with stencil test
	 */
	attachments := []core.AttachmentDescription{
		{
			Format:         info.Format,
			Samples:        common.Samples1,
			LoadOp:         common.LoadOpClear,
			StoreOp:        common.StoreOpStore,
			StencilLoadOp:  common.LoadOpDontCare,
			StencilStoreOp: common.StoreOpDontCare,
			InitialLayout:  common.LayoutUndefined,
			FinalLayout:    common.LayoutColorAttachmentOptimal,
		},
		{
			Format:         info.Depth.Format,
			Samples:        common.Samples1,
			LoadOp:         common.LoadOpClear,
			StoreOp:        common.StoreOpStore,
			StencilLoadOp:  common.LoadOpClear,
			StencilStoreOp: common.StoreOpStore,
			InitialLayout:  common.LayoutUndefined,
			FinalLayout:    common.LayoutDepthStencilAttachmentOptimal,
		},
	}
	depthStencilRef := &common.AttachmentReference{
		AttachmentIndex: 1,
		Layout:          common.LayoutDepthStencilAttachmentOptimal,
	}
	colorRef := common.AttachmentReference{
		AttachmentIndex: 0,
		Layout:          common.LayoutColorAttachmentOptimal,
	}

	subpass := core.SubPass{
		BindPoint:              common.BindGraphics,
		DepthStencilAttachment: depthStencilRef,
	}

	subpasses := []core.SubPass{}

	/* first a depthstencil-only subpass */
	subpasses = append(subpasses, subpass)

	subpass.ColorAttachments = []common.AttachmentReference{colorRef}

	/* then depthstencil and color */
	subpasses = append(subpasses, subpass)

	/* Set up a dependency between the source and destination subpasses */
	dependencies := []core.SubPassDependency{
		{
			SrcSubPassIndex: 0,
			DstSubPassIndex: 1,

			SrcStageMask: common.PipelineStageAllGraphics,
			DstStageMask: common.PipelineStageAllGraphics,

			SrcAccessMask: common.AccessDepthStencilAttachmentWrite | common.AccessDepthStencilAttachmentRead,
			DstAccessMask: common.AccessDepthStencilAttachmentWrite | common.AccessDepthStencilAttachmentRead,
		},
		{
			SrcSubPassIndex: core.SubpassExternal,
			DstSubPassIndex: 0,

			SrcStageMask: common.PipelineStageColorAttachmentOutput,
			DstStageMask: common.PipelineStageColorAttachmentOutput,

			SrcAccessMask: 0,
			DstAccessMask: common.AccessColorAttachmentWrite,
		},
	}

	renderPassOptions := &core.RenderPassOptions{
		Attachments:         attachments,
		SubPasses:           subpasses,
		SubPassDependencies: dependencies,
	}

	stencilRenderPass, _, err := info.Loader.CreateRenderPass(info.Device, nil, renderPassOptions)
	if err != nil {
		log.Fatalln(err)
	}

	/* now that we have the render pass, create framebuffer and pipelines */
	info.RenderPass = stencilRenderPass
	err = info.InitFramebuffers(true)
	if err != nil {
		log.Fatalln(err)
	}

	dynamicState := &core.DynamicStateOptions{
		DynamicStates: []core.DynamicState{},
	}

	vi := &core.VertexInputOptions{
		VertexBindingDescriptions:   []core.VertexBindingDescription{info.VertexBinding},
		VertexAttributeDescriptions: info.VertexAttributes,
	}

	ia := &core.InputAssemblyOptions{
		Topology: common.TopologyTriangleList,
	}

	rs := &core.RasterizationOptions{
		PolygonMode:             core.PolygonModeFill,
		CullMode:                common.CullBack,
		FrontFace:               common.FrontFaceClockwise,
		DepthClamp:              false,
		RasterizerDiscard:       false,
		DepthBias:               false,
		DepthBiasConstantFactor: 0,
		DepthBiasClamp:          0,
		DepthBiasSlopeFactor:    0,
		LineWidth:               1,
	}

	attState := []core.ColorBlendAttachment{
		{
			WriteMask:    0xf,
			BlendEnabled: false,
			AlphaBlendOp: common.BlendOpAdd,
			ColorBlendOp: common.BlendOpAdd,
			SrcColor:     common.BlendZero,
			DstColor:     common.BlendZero,
			SrcAlpha:     common.BlendZero,
			DstAlpha:     common.BlendZero,
		},
	}

	cb := &core.ColorBlendOptions{
		Attachments:    attState,
		LogicOpEnabled: false,
		LogicOp:        common.LogicOpNoop,
		BlendConstants: [4]float32{1, 1, 1, 1},
	}

	vp := &core.ViewportOptions{
		Viewports: []common.Viewport{
			{},
		},
		Scissors: []common.Rect2D{
			{},
		},
	}
	dynamicState.DynamicStates = append(dynamicState.DynamicStates, core.DynamicStateViewport, core.DynamicStateScissor)

	ds := &core.DepthStencilOptions{
		DepthTestEnable:       true,
		DepthWriteEnable:      true,
		DepthCompareOp:        common.CompareLessOrEqual,
		DepthBoundsTestEnable: false,
		MinDepthBounds:        0,
		MaxDepthBounds:        0,

		StencilTestEnable: true,
		BackStencilState: core.StencilOpState{
			FailOp:      common.StencilReplace,
			DepthFailOp: common.StencilReplace,
			PassOp:      common.StencilReplace,
			CompareMask: 0xff,
			WriteMask:   0xff,
			Reference:   0x44,
		},
	}
	ds.FrontStencilState = ds.BackStencilState

	ms := &core.MultisampleOptions{
		RasterizationSamples: utils.NumSamples,
		SampleShading:        false,
		MinSampleShading:     0,
		AlphaToCoverage:      false,
		AlphaToOne:           false,
	}

	pipelineOptions := &core.GraphicsPipelineOptions{
		Layout:            info.PipelineLayout,
		BasePipeline:      nil,
		BasePipelineIndex: 0,

		VertexInput:   vi,
		InputAssembly: ia,
		Rasterization: rs,
		Multisample:   ms,
		DynamicState:  dynamicState,
		Viewport:      vp,
		DepthStencil:  ds,
		RenderPass:    stencilRenderPass,
		SubPass:       0,
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
	pipelineOptions.ShaderStages = info.ShaderStages

	/* The first pipeline will render in subpass 0 to fill the stencil */
	pipelineOptions.SubPass = 0

	stencilCubePipe, _, err := info.Loader.CreateGraphicsPipelines(info.Device, info.PipelineCache, nil, []*core.GraphicsPipelineOptions{pipelineOptions})
	if err != nil {
		log.Fatalln(err)
	}

	/* destroy the shaders used for the above pipelin eand replace them with
	   those for the
	   fullscreen fill pass */
	info.DestroyShaders()
	fullscreenVertShaderBytes, err := fileSystem.ReadFile("shaders/full_vert.spv")
	if err != nil {
		log.Fatalln(err)
	}
	err = info.InitShaders(fullscreenVertShaderBytes, fragShaderBytes)
	if err != nil {
		log.Fatalln(err)
	}
	pipelineOptions.ShaderStages = info.ShaderStages

	/* the second pipeline will stencil test but not write, using the same
	 * reference */
	ds.BackStencilState.FailOp = common.StencilKeep
	ds.BackStencilState.DepthFailOp = common.StencilKeep
	ds.BackStencilState.PassOp = common.StencilKeep
	ds.BackStencilState.CompareOp = common.CompareEqual
	ds.FrontStencilState = ds.BackStencilState

	/* don't test depth, only use stencil test */
	ds.DepthTestEnable = false

	/* the second pipeline will be a fullscreen triangle strip, with vertices
	   generated purely from the vertex shader - no inputs needed */
	ia.Topology = common.TopologyTriangleStrip
	vi.VertexBindingDescriptions = nil
	vi.VertexAttributeDescriptions = nil

	/* this pipeline will run in the second subpass */
	pipelineOptions.SubPass = 1
	pipelineOptions.ColorBlend = cb

	stencilFullscreenPipe, _, err := info.Loader.CreateGraphicsPipelines(info.Device, info.PipelineCache, nil, []*core.GraphicsPipelineOptions{pipelineOptions})
	if err != nil {
		log.Fatalln(err)
	}

	info.DestroyShaders()
	info.Pipeline = nil

	clearValues := []core.ClearValue{
		core.ClearValueFloat{0.2, 0.2, 0.2, 0.2},
		core.ClearValueDepthStencil{Depth: 1.0, Stencil: 0},
	}

	imageAcquiredSemaphore, _, err := info.Loader.CreateSemaphore(info.Device, nil, &core.SemaphoreOptions{})
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

	/* Begin the first render pass. This will render in the left half of the
	   screen. Subpass 0 will render a cube, stencil writing but outputting
	   no color. Subpass 1 will render a fullscreen pass, stencil testing and
	   outputting color only where the cube filled in stencil */
	renderPassBegin := &core.RenderPassBeginOptions{
		RenderPass:  stencilRenderPass,
		Framebuffer: info.Framebuffer[info.CurrentBuffer],
		RenderArea: common.Rect2D{
			Offset: common.Offset2D{0, 0},
			Extent: common.Extent2D{info.Width / 2, info.Height},
		},
		ClearValues: clearValues,
	}
	info.Cmd.CmdBeginRenderPass(core.ContentsInline, renderPassBegin)

	info.Cmd.CmdBindPipeline(common.BindGraphics, stencilCubePipe[0])
	info.Cmd.CmdBindDescriptorSets(common.BindGraphics, info.PipelineLayout, info.DescSet, nil)
	info.Cmd.CmdBindVertexBuffers([]core.Buffer{info.VertexBuffer.Buf}, []int{0})

	viewports := []common.Viewport{
		{
			X:        0,
			Y:        0,
			Width:    float32(info.Width) / 2.0,
			Height:   float32(info.Height),
			MinDepth: 0,
			MaxDepth: 1,
		},
	}
	info.Cmd.CmdSetViewport(viewports)

	scissors := []common.Rect2D{
		{
			Offset: common.Offset2D{0, 0},
			Extent: common.Extent2D{info.Width / 2, info.Height},
		},
	}
	info.Cmd.CmdSetScissor(scissors)

	/* Draw the cube into stencil */
	info.Cmd.CmdDraw(36, 1, 0, 0)

	/* Advance to the next subpass */
	info.Cmd.CmdNextSubpass(core.ContentsInline)

	/* Bind the fullscreen pass pipeline */
	info.Cmd.CmdBindPipeline(common.BindGraphics, stencilFullscreenPipe[0])

	info.Cmd.CmdSetViewport(viewports)
	info.Cmd.CmdSetScissor(scissors)

	/* Draw the fullscreen pass */
	info.Cmd.CmdDraw(4, 1, 0, 0)
	info.Cmd.CmdEndRenderPass()

	/**
	 * Second renderpass in this sample.
	 * Blended rendering, each subpass blends continuously onto the color
	 */

	/* note that we reuse a lot of the initialisation strutures from the first
	   render pass, so this represents a 'delta' from that configuration */
	renderPassOptions.SubPasses[0].ColorAttachments = []common.AttachmentReference{
		colorRef,
	}
	renderPassOptions.Attachments[0].InitialLayout = common.LayoutColorAttachmentOptimal
	renderPassOptions.Attachments[0].FinalLayout = common.LayoutPresentSrcKHR
	renderPassOptions.Attachments[1].InitialLayout = common.LayoutDepthStencilAttachmentOptimal

	renderPassOptions.SubPassDependencies[0].SrcAccessMask |= common.AccessColorAttachmentWrite | common.AccessColorAttachmentRead
	renderPassOptions.SubPassDependencies[0].DstAccessMask |= common.AccessColorAttachmentRead | common.AccessColorAttachmentWrite
	renderPassOptions.SubPassDependencies = renderPassOptions.SubPassDependencies[0:1]

	blendRenderPass, _, err := info.Loader.CreateRenderPass(info.Device, nil, renderPassOptions)
	if err != nil {
		log.Fatalln(err)
	}

	pipelineOptions.RenderPass = blendRenderPass

	/* We must recreate the framebuffers with this renderpass as the two render
	   passes are not compatible. Store the current framebuffers for later
	   deletion */
	stencilFramebuffers := info.Framebuffer
	info.Framebuffer = nil

	info.RenderPass = blendRenderPass
	err = info.InitFramebuffers(true)
	if err != nil {
		log.Fatalln(err)
	}

	/* Now create the pipelines for the second render pass */

	/* We are rendering the cube again, configure the vertex inputs */
	ia.Topology = common.TopologyTriangleList
	vi.VertexBindingDescriptions = []core.VertexBindingDescription{info.VertexBinding}
	vi.VertexAttributeDescriptions = info.VertexAttributes

	/* The first pipeline will depth write and depth test */
	ds.DepthWriteEnable = true
	ds.DepthTestEnable = true

	/* We don't want to stencil test */
	ds.StencilTestEnable = false

	/* This time, both pipelines will blend. the first pipeline uses the blend
	   constant
	   to determine the blend amount */
	cb.Attachments[0].WriteMask = 0xf
	cb.Attachments[0].BlendEnabled = true
	cb.Attachments[0].AlphaBlendOp = common.BlendOpAdd
	cb.Attachments[0].ColorBlendOp = common.BlendOpAdd
	cb.Attachments[0].SrcColor = common.BlendConstantAlpha
	cb.Attachments[0].DstColor = common.BlendOne
	cb.Attachments[0].SrcAlpha = common.BlendConstantAlpha
	cb.Attachments[0].DstAlpha = common.BlendOne
	cb.BlendConstants = [4]float32{1, 1, 1, 0.3}

	err = info.InitShaders(vertShaderBytes, fragShaderBytes)
	if err != nil {
		log.Fatalln(err)
	}
	pipelineOptions.ShaderStages = info.ShaderStages

	/* This is the first subpass's pipeline, to blend a cube onto the color
	 * image */
	pipelineOptions.SubPass = 0

	blendCubePipe, _, err := info.Loader.CreateGraphicsPipelines(info.Device, info.PipelineCache, nil, []*core.GraphicsPipelineOptions{
		pipelineOptions,
	})
	if err != nil {
		log.Fatalln(err)
	}

	/* Now we will set up the fullscreen pass to render on top. */
	info.DestroyShaders()
	err = info.InitShaders(fullscreenVertShaderBytes, fragShaderBytes)
	if err != nil {
		log.Fatalln(err)
	}
	pipelineOptions.ShaderStages = info.ShaderStages

	/* the second pipeline will be a fullscreen triangle strip with no inputs */
	ia.Topology = common.TopologyTriangleStrip
	vi.VertexBindingDescriptions = nil
	vi.VertexAttributeDescriptions = nil

	/* We'll use the alpha output from the shader */
	cb.Attachments[0].SrcColor = common.BlendSrcAlpha
	cb.Attachments[0].DstColor = common.BlendOne
	cb.Attachments[0].SrcAlpha = common.BlendSrcAlpha
	cb.Attachments[0].DstAlpha = common.BlendOne

	/* This renders in the second subpass */
	pipelineOptions.SubPass = 1

	blendFullscreenPipe, _, err := info.Loader.CreateGraphicsPipelines(info.Device, info.PipelineCache, nil, []*core.GraphicsPipelineOptions{pipelineOptions})
	if err != nil {
		log.Fatalln(err)
	}

	info.DestroyShaders()
	info.Pipeline = nil

	/* Now we are going to render in the right half of the screen */
	viewports[0].X = float32(info.Width) / 2.0
	scissors[0].Offset.X = info.Width / 2
	renderPassBegin.RenderArea.Offset.X = info.Width / 2

	/* Use our framebuffer and render pass */
	renderPassBegin.Framebuffer = info.Framebuffer[info.CurrentBuffer]
	renderPassBegin.RenderPass = blendRenderPass
	err = info.Cmd.CmdBeginRenderPass(core.ContentsInline, renderPassBegin)
	if err != nil {
		log.Fatalln(err)
	}

	info.Cmd.CmdBindPipeline(common.BindGraphics, blendCubePipe[0])
	info.Cmd.CmdBindDescriptorSets(common.BindGraphics, info.PipelineLayout, info.DescSet, nil)
	info.Cmd.CmdBindVertexBuffers([]core.Buffer{info.VertexBuffer.Buf}, []int{0})
	info.Cmd.CmdSetViewport(viewports)
	info.Cmd.CmdSetScissor(scissors)

	/* Draw the cube blending */
	info.Cmd.CmdDraw(36, 1, 0, 0)

	/* Advance to the next subpass */
	info.Cmd.CmdNextSubpass(core.ContentsInline)

	info.Cmd.CmdBindPipeline(common.BindGraphics, blendFullscreenPipe[0])

	/* Adjust the viewport to be a square in the centre, just overlapping the
	 * cube */
	viewports[0].X += 25
	viewports[0].Y += 150
	viewports[0].Width -= 50
	viewports[0].Height -= 300

	info.Cmd.CmdSetViewport(viewports)
	info.Cmd.CmdSetScissor(scissors)
	info.Cmd.CmdDraw(4, 1, 0, 0)

	/* The second renderpass is complete */
	info.Cmd.CmdEndRenderPass()

	/* VULKAN_KEY_END */

	_, err = info.Cmd.End()
	if err != nil {
		log.Fatalln(err)
	}
	drawFence, _, err := info.Loader.CreateFence(info.Device, nil, &core.FenceOptions{})
	if err != nil {
		log.Fatalln(err)
	}

	/* Queue the command buffer for execution */
	_, err = info.GraphicsQueue.SubmitToQueue(drawFence, []*core.SubmitOptions{
		{
			WaitSemaphores: []core.Semaphore{imageAcquiredSemaphore},
			CommandBuffers: []core.CommandBuffer{info.Cmd},
			WaitDstStages:  []common.PipelineStages{common.PipelineStageColorAttachmentOutput},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	/* Now present the image in the window */

	/* Make sure command buffer is finished before presenting */
	for {
		res, err := info.Device.WaitForFences(true, utils.FenceTimeout, []core.Fence{drawFence})
		if err != nil {
			log.Fatalln(err)
		}

		if res != core.VKTimeout {
			break
		}
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
		err = info.WritePNG("draw_subpasses")
		if err != nil {
			log.Fatalln(err)
		}
	}

	for i := 0; i < info.SwapchainImageCount; i++ {
		stencilFramebuffers[i].Destroy()
	}

	stencilRenderPass.Destroy()
	blendRenderPass.Destroy()

	blendCubePipe[0].Destroy()
	blendFullscreenPipe[0].Destroy()

	stencilCubePipe[0].Destroy()
	stencilFullscreenPipe[0].Destroy()

	imageAcquiredSemaphore.Destroy()
	drawFence.Destroy()
	info.DestroyPipelineCache()
	info.DestroyDescriptorPool()
	info.DestroyVertexBuffer()
	info.DestroyFramebuffers()
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
	info.Window.Destroy()
}
