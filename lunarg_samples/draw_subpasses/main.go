package main

import (
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
	debugOptions := ext_debug_utils.CreateOptions{
		CaptureSeverities: ext_debug_utils.SeverityWarning | ext_debug_utils.SeverityError,
		CaptureTypes:      ext_debug_utils.TypeGeneral | ext_debug_utils.TypeValidation | ext_debug_utils.TypePerformance,
		Callback:          logDebug,
	}

	err = info.InitInstance("Multi-pass render passes", debugOptions)
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

	err = info.InitSwapchain(core1_0.ImageUsageColorAttachment | core1_0.ImageUsageTransferSrc)
	if err != nil {
		log.Fatalln(err)
	}

	props := info.Gpus[0].FormatProperties(core1_0.DataFormatD32SignedFloatS8UnsignedInt)
	if (props.LinearTilingFeatures&core1_0.FormatFeatureDepthStencilAttachment != 0) ||
		(props.OptimalTilingFeatures&core1_0.FormatFeatureDepthStencilAttachment != 0) {
		info.Depth.Format = core1_0.DataFormatD32SignedFloatS8UnsignedInt
	} else {
		info.Depth.Format = core1_0.DataFormatD24UnsignedNormalizedS8UnsignedInt
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
	attachments := []core1_0.AttachmentDescription{
		{
			Format:         info.Format,
			Samples:        core1_0.Samples1,
			LoadOp:         core1_0.LoadOpClear,
			StoreOp:        core1_0.StoreOpStore,
			StencilLoadOp:  core1_0.LoadOpDontCare,
			StencilStoreOp: core1_0.StoreOpDontCare,
			InitialLayout:  core1_0.ImageLayoutUndefined,
			FinalLayout:    core1_0.ImageLayoutColorAttachmentOptimal,
		},
		{
			Format:         info.Depth.Format,
			Samples:        core1_0.Samples1,
			LoadOp:         core1_0.LoadOpClear,
			StoreOp:        core1_0.StoreOpStore,
			StencilLoadOp:  core1_0.LoadOpClear,
			StencilStoreOp: core1_0.StoreOpStore,
			InitialLayout:  core1_0.ImageLayoutUndefined,
			FinalLayout:    core1_0.ImageLayoutDepthStencilAttachmentOptimal,
		},
	}
	depthStencilRef := &core1_0.AttachmentReference{
		AttachmentIndex: 1,
		Layout:          core1_0.ImageLayoutDepthStencilAttachmentOptimal,
	}
	colorRef := core1_0.AttachmentReference{
		AttachmentIndex: 0,
		Layout:          core1_0.ImageLayoutColorAttachmentOptimal,
	}
	unusedRef := core1_0.AttachmentReference{
		AttachmentIndex: -1,
		Layout:          core1_0.ImageLayoutUndefined,
	}

	subpass := core1_0.SubPassDescription{
		BindPoint:              core1_0.BindGraphics,
		DepthStencilAttachment: depthStencilRef,
		ColorAttachments: []core1_0.AttachmentReference{
			unusedRef,
		},
	}

	subpasses := []core1_0.SubPassDescription{}

	/* first a depthstencil-only subpass */
	subpasses = append(subpasses, subpass)

	subpass.ColorAttachments = []core1_0.AttachmentReference{colorRef}

	/* then depthstencil and color */
	subpasses = append(subpasses, subpass)

	/* Set up a dependency between the source and destination subpasses */
	dependencies := []core1_0.SubPassDependency{
		{
			SrcSubPassIndex: 0,
			DstSubPassIndex: 1,

			SrcStageMask: core1_0.PipelineStageAllGraphics,
			DstStageMask: core1_0.PipelineStageAllGraphics,

			SrcAccessMask: core1_0.AccessDepthStencilAttachmentWrite | core1_0.AccessDepthStencilAttachmentRead,
			DstAccessMask: core1_0.AccessDepthStencilAttachmentWrite | core1_0.AccessDepthStencilAttachmentRead,
		},
		{
			SrcSubPassIndex: core1_0.SubpassExternal,
			DstSubPassIndex: 0,

			SrcStageMask: core1_0.PipelineStageColorAttachmentOutput,
			DstStageMask: core1_0.PipelineStageColorAttachmentOutput,

			SrcAccessMask: 0,
			DstAccessMask: core1_0.AccessColorAttachmentWrite,
		},
	}

	renderPassOptions := core1_0.RenderPassCreateOptions{
		Attachments:         attachments,
		SubPassDescriptions: subpasses,
		SubPassDependencies: dependencies,
	}

	stencilRenderPass, _, err := info.Device.CreateRenderPass(nil, renderPassOptions)
	if err != nil {
		log.Fatalln(err)
	}

	/* now that we have the render pass, create framebuffer and pipelines */
	info.RenderPass = stencilRenderPass
	err = info.InitFramebuffers(true)
	if err != nil {
		log.Fatalln(err)
	}

	dynamicState := &core1_0.DynamicStateOptions{
		DynamicStates: []core1_0.DynamicState{},
	}

	vi := &core1_0.VertexInputStateOptions{
		VertexBindingDescriptions:   []core1_0.VertexBindingDescription{info.VertexBinding},
		VertexAttributeDescriptions: info.VertexAttributes,
	}

	ia := &core1_0.InputAssemblyStateOptions{
		Topology: core1_0.TopologyTriangleList,
	}

	rs := &core1_0.RasterizationStateOptions{
		PolygonMode:             core1_0.PolygonModeFill,
		CullMode:                core1_0.CullBack,
		FrontFace:               core1_0.FrontFaceClockwise,
		DepthClamp:              false,
		RasterizerDiscard:       false,
		DepthBias:               false,
		DepthBiasConstantFactor: 0,
		DepthBiasClamp:          0,
		DepthBiasSlopeFactor:    0,
		LineWidth:               1,
	}

	attState := []core1_0.ColorBlendAttachment{
		{
			WriteMask:    0xf,
			BlendEnabled: false,
			AlphaBlendOp: core1_0.BlendOpAdd,
			ColorBlendOp: core1_0.BlendOpAdd,
			SrcColor:     core1_0.BlendZero,
			DstColor:     core1_0.BlendZero,
			SrcAlpha:     core1_0.BlendZero,
			DstAlpha:     core1_0.BlendZero,
		},
	}

	cb := &core1_0.ColorBlendStateOptions{
		Attachments:    attState,
		LogicOpEnabled: false,
		LogicOp:        core1_0.LogicOpNoop,
		BlendConstants: [4]float32{1, 1, 1, 1},
	}

	vp := &core1_0.ViewportStateOptions{
		Viewports: []core1_0.Viewport{
			{},
		},
		Scissors: []core1_0.Rect2D{
			{},
		},
	}
	dynamicState.DynamicStates = append(dynamicState.DynamicStates, core1_0.DynamicStateViewport, core1_0.DynamicStateScissor)

	ds := &core1_0.DepthStencilStateOptions{
		DepthTestEnable:       true,
		DepthWriteEnable:      true,
		DepthCompareOp:        core1_0.CompareLessOrEqual,
		DepthBoundsTestEnable: false,
		MinDepthBounds:        0,
		MaxDepthBounds:        0,

		StencilTestEnable: true,
		BackStencilState: core1_0.StencilOpState{
			FailOp:      core1_0.StencilReplace,
			DepthFailOp: core1_0.StencilReplace,
			PassOp:      core1_0.StencilReplace,
			CompareMask: 0xff,
			WriteMask:   0xff,
			Reference:   0x44,
		},
	}
	ds.FrontStencilState = ds.BackStencilState

	ms := &core1_0.MultisampleStateOptions{
		RasterizationSamples: utils.NumSamples,
		SampleShading:        false,
		MinSampleShading:     0,
		AlphaToCoverage:      false,
		AlphaToOne:           false,
	}

	pipelineOptions := core1_0.GraphicsPipelineCreateOptions{
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

	stencilCubePipe, _, err := info.Device.CreateGraphicsPipelines(info.PipelineCache, nil, []core1_0.GraphicsPipelineCreateOptions{pipelineOptions})
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
	ds.BackStencilState.FailOp = core1_0.StencilKeep
	ds.BackStencilState.DepthFailOp = core1_0.StencilKeep
	ds.BackStencilState.PassOp = core1_0.StencilKeep
	ds.BackStencilState.CompareOp = core1_0.CompareEqual
	ds.FrontStencilState = ds.BackStencilState

	/* don't test depth, only use stencil test */
	ds.DepthTestEnable = false

	/* the second pipeline will be a fullscreen triangle strip, with vertices
	   generated purely from the vertex shader - no inputs needed */
	ia.Topology = core1_0.TopologyTriangleStrip
	vi.VertexBindingDescriptions = nil
	vi.VertexAttributeDescriptions = nil

	/* this pipeline will run in the second subpass */
	pipelineOptions.SubPass = 1
	pipelineOptions.ColorBlend = cb

	stencilFullscreenPipe, _, err := info.Device.CreateGraphicsPipelines(info.PipelineCache, nil, []core1_0.GraphicsPipelineCreateOptions{pipelineOptions})
	if err != nil {
		log.Fatalln(err)
	}

	info.DestroyShaders()
	info.Pipeline = nil

	clearValues := []core1_0.ClearValue{
		core1_0.ClearValueFloat{0.2, 0.2, 0.2, 0.2},
		core1_0.ClearValueDepthStencil{Depth: 1.0, Stencil: 0},
	}

	imageAcquiredSemaphore, _, err := info.Device.CreateSemaphore(nil, core1_0.SemaphoreCreateOptions{})
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
	renderPassBegin := core1_0.RenderPassBeginOptions{
		RenderPass:  stencilRenderPass,
		Framebuffer: info.Framebuffer[info.CurrentBuffer],
		RenderArea: core1_0.Rect2D{
			Offset: core1_0.Offset2D{0, 0},
			Extent: core1_0.Extent2D{info.Width / 2, info.Height},
		},
		ClearValues: clearValues,
	}
	info.Cmd.CmdBeginRenderPass(core1_0.SubpassContentsInline, renderPassBegin)

	info.Cmd.CmdBindPipeline(core1_0.BindGraphics, stencilCubePipe[0])
	info.Cmd.CmdBindDescriptorSets(core1_0.BindGraphics, info.PipelineLayout, info.DescSet, nil)
	info.Cmd.CmdBindVertexBuffers([]core1_0.Buffer{info.VertexBuffer.Buf}, []int{0})

	viewports := []core1_0.Viewport{
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

	scissors := []core1_0.Rect2D{
		{
			Offset: core1_0.Offset2D{0, 0},
			Extent: core1_0.Extent2D{info.Width / 2, info.Height},
		},
	}
	info.Cmd.CmdSetScissor(scissors)

	/* Draw the cube into stencil */
	info.Cmd.CmdDraw(36, 1, 0, 0)

	/* Advance to the next subpass */
	info.Cmd.CmdNextSubpass(core1_0.SubpassContentsInline)

	/* Bind the fullscreen pass pipeline */
	info.Cmd.CmdBindPipeline(core1_0.BindGraphics, stencilFullscreenPipe[0])

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
	renderPassOptions.SubPassDescriptions[0].ColorAttachments[0] = colorRef
	renderPassOptions.Attachments[0].InitialLayout = core1_0.ImageLayoutColorAttachmentOptimal
	renderPassOptions.Attachments[0].FinalLayout = khr_swapchain.ImageLayoutPresentSrc
	renderPassOptions.Attachments[1].InitialLayout = core1_0.ImageLayoutDepthStencilAttachmentOptimal

	renderPassOptions.SubPassDependencies[0].SrcAccessMask |= core1_0.AccessColorAttachmentWrite | core1_0.AccessColorAttachmentRead
	renderPassOptions.SubPassDependencies[0].DstAccessMask |= core1_0.AccessColorAttachmentRead | core1_0.AccessColorAttachmentWrite
	renderPassOptions.SubPassDependencies = renderPassOptions.SubPassDependencies[0:1]

	blendRenderPass, _, err := info.Device.CreateRenderPass(nil, renderPassOptions)
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
	ia.Topology = core1_0.TopologyTriangleList
	vi.VertexBindingDescriptions = []core1_0.VertexBindingDescription{info.VertexBinding}
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
	cb.Attachments[0].AlphaBlendOp = core1_0.BlendOpAdd
	cb.Attachments[0].ColorBlendOp = core1_0.BlendOpAdd
	cb.Attachments[0].SrcColor = core1_0.BlendConstantAlpha
	cb.Attachments[0].DstColor = core1_0.BlendOne
	cb.Attachments[0].SrcAlpha = core1_0.BlendConstantAlpha
	cb.Attachments[0].DstAlpha = core1_0.BlendOne
	cb.BlendConstants = [4]float32{1, 1, 1, 0.3}

	err = info.InitShaders(vertShaderBytes, fragShaderBytes)
	if err != nil {
		log.Fatalln(err)
	}
	pipelineOptions.ShaderStages = info.ShaderStages

	/* This is the first subpass's pipeline, to blend a cube onto the color
	 * image */
	pipelineOptions.SubPass = 0

	blendCubePipe, _, err := info.Device.CreateGraphicsPipelines(info.PipelineCache, nil, []core1_0.GraphicsPipelineCreateOptions{
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
	ia.Topology = core1_0.TopologyTriangleStrip
	vi.VertexBindingDescriptions = nil
	vi.VertexAttributeDescriptions = nil

	/* We'll use the alpha output from the shader */
	cb.Attachments[0].SrcColor = core1_0.BlendSrcAlpha
	cb.Attachments[0].DstColor = core1_0.BlendOne
	cb.Attachments[0].SrcAlpha = core1_0.BlendSrcAlpha
	cb.Attachments[0].DstAlpha = core1_0.BlendOne

	/* This renders in the second subpass */
	pipelineOptions.SubPass = 1

	blendFullscreenPipe, _, err := info.Device.CreateGraphicsPipelines(info.PipelineCache, nil, []core1_0.GraphicsPipelineCreateOptions{pipelineOptions})
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
	err = info.Cmd.CmdBeginRenderPass(core1_0.SubpassContentsInline, renderPassBegin)
	if err != nil {
		log.Fatalln(err)
	}

	info.Cmd.CmdBindPipeline(core1_0.BindGraphics, blendCubePipe[0])
	info.Cmd.CmdBindDescriptorSets(core1_0.BindGraphics, info.PipelineLayout, info.DescSet, nil)
	info.Cmd.CmdBindVertexBuffers([]core1_0.Buffer{info.VertexBuffer.Buf}, []int{0})
	info.Cmd.CmdSetViewport(viewports)
	info.Cmd.CmdSetScissor(scissors)

	/* Draw the cube blending */
	info.Cmd.CmdDraw(36, 1, 0, 0)

	/* Advance to the next subpass */
	info.Cmd.CmdNextSubpass(core1_0.SubpassContentsInline)

	info.Cmd.CmdBindPipeline(core1_0.BindGraphics, blendFullscreenPipe[0])

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
	drawFence, _, err := info.Device.CreateFence(nil, core1_0.FenceCreateOptions{})
	if err != nil {
		log.Fatalln(err)
	}

	/* Queue the command buffer for execution */
	_, err = info.GraphicsQueue.SubmitToQueue(drawFence, []core1_0.SubmitOptions{
		{
			WaitSemaphores: []core1_0.Semaphore{imageAcquiredSemaphore},
			CommandBuffers: []core1_0.CommandBuffer{info.Cmd},
			WaitDstStages:  []core1_0.PipelineStages{core1_0.PipelineStageColorAttachmentOutput},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	/* Now present the image in the window */

	/* Make sure command buffer is finished before presenting */
	for {
		res, err := info.Device.WaitForFences(true, utils.FenceTimeout, []core1_0.Fence{drawFence})
		if err != nil {
			log.Fatalln(err)
		}

		if res != core1_0.VKTimeout {
			break
		}
	}
	_, err = info.SwapchainExtension.PresentToQueue(info.PresentQueue, khr_swapchain.PresentOptions{
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
		stencilFramebuffers[i].Destroy(nil)
	}

	stencilRenderPass.Destroy(nil)
	blendRenderPass.Destroy(nil)

	blendCubePipe[0].Destroy(nil)
	blendFullscreenPipe[0].Destroy(nil)

	stencilCubePipe[0].Destroy(nil)
	stencilFullscreenPipe[0].Destroy(nil)

	imageAcquiredSemaphore.Destroy(nil)
	drawFence.Destroy(nil)
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

	info.Surface.Destroy(nil)
	debugMessenger.Destroy(nil)
	info.DestroyInstance()
	info.Window.Destroy()
}
