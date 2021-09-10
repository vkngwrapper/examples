package main

import (
	"embed"
	"fmt"
	"github.com/CannibalVox/VKng/core"
	commands2 "github.com/CannibalVox/VKng/core/commands"
	"github.com/CannibalVox/VKng/core/loader"
	pipeline2 "github.com/CannibalVox/VKng/core/pipeline"
	render_pass2 "github.com/CannibalVox/VKng/core/render_pass"
	"github.com/CannibalVox/VKng/core/resource"
	ext_debugutils2 "github.com/CannibalVox/VKng/extensions/debugutils"
	ext_surface2 "github.com/CannibalVox/VKng/extensions/surface"
	ext_surface_sdl22 "github.com/CannibalVox/VKng/extensions/surface_sdl"
	ext_swapchain2 "github.com/CannibalVox/VKng/extensions/swapchain"
	"github.com/CannibalVox/cgoalloc"
	"github.com/cockroachdb/errors"
	"github.com/veandco/go-sdl2/sdl"
	"log"
	"unsafe"
)

//go:embed shaders
var shaders embed.FS

const MaxFramesInFlight = 2

var validationLayers = []string{"VK_LAYER_KHRONOS_validation"}
var deviceExtensions = []string{ext_swapchain2.ExtensionName}

const enableValidationLayers = true

type QueueFamilyIndices struct {
	GraphicsFamily *int
	PresentFamily  *int
}

func (i *QueueFamilyIndices) IsComplete() bool {
	return i.GraphicsFamily != nil && i.PresentFamily != nil
}

type SwapChainSupportDetails struct {
	Capabilities *ext_surface2.Capabilities
	Formats      []ext_surface2.Format
	PresentModes []ext_surface2.PresentMode
}

type Vertex struct {
	X, Y    float32 // Could also be Position Vector2 - as long as Vector2 is a value, not a pointer
	R, G, B float32 // Could also be Color Vector3
}

func getVertexBindingDescription() []pipeline2.VertexBindingDescription {
	v := Vertex{}
	return []pipeline2.VertexBindingDescription{
		{
			Binding:   0,
			Stride:    unsafe.Sizeof(v),
			InputRate: pipeline2.RateVertex,
		},
	}
}

func getVertexAttributeDescriptions() []pipeline2.VertexAttributeDescription {
	v := Vertex{}
	return []pipeline2.VertexAttributeDescription{
		{
			Binding:  0,
			Location: 0,
			Format:   core.FormatR32G32SignedFloat,
			Offset:   unsafe.Offsetof(v.X),
		},
		{
			Binding:  0,
			Location: 1,
			Format:   core.FormatR32G32B32SignedFloat,
			Offset:   unsafe.Offsetof(v.R),
		},
	}
}

var vertices = []Vertex{
	{X: 0, Y: -0.5, R: 1, G: 0, B: 0},
	{X: 0.5, Y: 0.5, R: 0, G: 1, B: 0},
	{X: -0.5, Y: 0.5, R: 0, G: 0, B: 1},
}

type HelloTriangleApplication struct {
	allocator cgoalloc.Allocator
	window    *sdl.Window
	loader    *loader.Loader

	instance       *resource.Instance
	debugMessenger *ext_debugutils2.Messenger
	surface        *ext_surface2.Surface

	physicalDevice *resource.PhysicalDevice
	device         *resource.Device

	graphicsQueue *resource.Queue
	presentQueue  *resource.Queue

	swapchain             *ext_swapchain2.Swapchain
	swapchainImages       []*resource.Image
	swapchainImageFormat  core.DataFormat
	swapchainExtent       core.Extent2D
	swapchainImageViews   []*resource.ImageView
	swapchainFramebuffers []*render_pass2.Framebuffer

	renderPass       *render_pass2.RenderPass
	pipelineLayout   *pipeline2.PipelineLayout
	graphicsPipeline *pipeline2.Pipeline

	commandPool    *commands2.CommandPool
	commandBuffers []*commands2.CommandBuffer

	imageAvailableSemaphore []*resource.Semaphore
	renderFinishedSemaphore []*resource.Semaphore
	inFlightFence           []*resource.Fence
	imagesInFlight          []*resource.Fence
	currentFrame            int
}

func (app *HelloTriangleApplication) Run() error {
	err := app.initWindow()
	if err != nil {
		return err
	}

	err = app.initVulkan()
	if err != nil {
		return err
	}
	defer app.cleanup()

	return app.mainLoop()
}

func (app *HelloTriangleApplication) initWindow() error {
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		return err
	}

	window, err := sdl.CreateWindow("Vulkan", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, 800, 600, sdl.WINDOW_SHOWN|sdl.WINDOW_VULKAN|sdl.WINDOW_RESIZABLE)
	if err != nil {
		return err
	}
	app.window = window

	app.loader, err = loader.CreateLoaderFromProcAddr(sdl.VulkanGetVkGetInstanceProcAddr())
	if err != nil {
		return err
	}

	return nil
}

func (app *HelloTriangleApplication) initVulkan() error {
	err := app.createInstance()
	if err != nil {
		return err
	}

	err = app.setupDebugMessenger()
	if err != nil {
		return err
	}

	err = app.createSurface()
	if err != nil {
		return err
	}

	err = app.pickPhysicalDevice()
	if err != nil {
		return err
	}

	err = app.createLogicalDevice()
	if err != nil {
		return err
	}

	err = app.createSwapchain()
	if err != nil {
		return err
	}

	err = app.createImageViews()
	if err != nil {
		return err
	}

	err = app.createRenderPass()
	if err != nil {
		return err
	}

	err = app.createGraphicsPipeline()
	if err != nil {
		return err
	}

	err = app.createFramebuffers()
	if err != nil {
		return err
	}

	err = app.createCommandPool()
	if err != nil {
		return err
	}

	err = app.createCommandBuffers()
	if err != nil {
		return err
	}

	return app.createSyncObjects()
}

func (app *HelloTriangleApplication) mainLoop() error {
	rendering := true

appLoop:
	for true {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch e := event.(type) {
			case *sdl.QuitEvent:
				break appLoop
			case *sdl.WindowEvent:
				switch e.Event {
				case sdl.WINDOWEVENT_MINIMIZED:
					rendering = false
				case sdl.WINDOWEVENT_RESTORED:
					rendering = true
				case sdl.WINDOWEVENT_RESIZED:
					w, h := app.window.GetSize()
					if w > 0 && h > 0 {
						rendering = true
					} else {
						rendering = false
					}
				}
			}
		}
		if rendering {
			err := app.drawFrame()
			if err != nil {
				return err
			}
		}
	}

	_, err := app.device.WaitForIdle()
	return err
}

func (app *HelloTriangleApplication) cleanupSwapChain() {
	for _, framebuffer := range app.swapchainFramebuffers {
		framebuffer.Destroy()
	}
	app.swapchainFramebuffers = []*render_pass2.Framebuffer{}

	if len(app.commandBuffers) > 0 {
		commands2.DestroyBuffers(app.allocator, app.commandPool, app.commandBuffers)
		app.commandBuffers = []*commands2.CommandBuffer{}
	}

	if app.graphicsPipeline != nil {
		app.graphicsPipeline.Destroy()
		app.graphicsPipeline = nil
	}

	if app.pipelineLayout != nil {
		app.pipelineLayout.Destroy()
		app.pipelineLayout = nil
	}

	if app.renderPass != nil {
		app.renderPass.Destroy()
		app.renderPass = nil
	}

	for _, imageView := range app.swapchainImageViews {
		imageView.Destroy()
	}
	app.swapchainImageViews = []*resource.ImageView{}

	if app.swapchain != nil {
		app.swapchain.Destroy()
		app.swapchain = nil
	}
}

func (app *HelloTriangleApplication) cleanup() {
	app.cleanupSwapChain()

	for _, fence := range app.inFlightFence {
		fence.Destroy()
	}

	for _, semaphore := range app.renderFinishedSemaphore {
		semaphore.Destroy()
	}

	for _, semaphore := range app.imageAvailableSemaphore {
		semaphore.Destroy()
	}

	if app.commandPool != nil {
		app.commandPool.Destroy()
	}

	if app.device != nil {
		app.device.Destroy()
	}

	if app.debugMessenger != nil {
		app.debugMessenger.Destroy()
	}

	if app.surface != nil {
		app.surface.Destroy()
	}

	if app.instance != nil {
		app.instance.Destroy()
	}

	if app.window != nil {
		app.window.Destroy()
	}
	sdl.Quit()

	app.allocator.Destroy()
}

func (app *HelloTriangleApplication) recreateSwapChain() error {
	fmt.Println("Recreating swap chain")
	w, h := app.window.VulkanGetDrawableSize()
	if w == 0 || h == 0 {
		return nil
	}
	if (app.window.GetFlags() & sdl.WINDOW_MINIMIZED) != 0 {
		return nil
	}

	_, err := app.device.WaitForIdle()
	if err != nil {
		return err
	}

	app.cleanupSwapChain()

	err = app.createSwapchain()
	if err != nil {
		return err
	}

	err = app.createImageViews()
	if err != nil {
		return err
	}

	err = app.createRenderPass()
	if err != nil {
		return err
	}

	err = app.createGraphicsPipeline()
	if err != nil {
		return err
	}

	err = app.createFramebuffers()
	if err != nil {
		return err
	}

	err = app.createCommandBuffers()
	if err != nil {
		return err
	}

	app.imagesInFlight = []*resource.Fence{}
	for i := 0; i < len(app.swapchainImages); i++ {
		app.imagesInFlight = append(app.imagesInFlight, nil)
	}

	return nil
}

func (app *HelloTriangleApplication) createInstance() error {
	instanceOptions := &resource.InstanceOptions{
		ApplicationName:    "Hello Triangle",
		ApplicationVersion: core.CreateVersion(1, 0, 0),
		EngineName:         "No Engine",
		EngineVersion:      core.CreateVersion(1, 0, 0),
		VulkanVersion:      core.Vulkan1_2,
	}

	// Add extensions
	sdlExtensions := app.window.VulkanGetInstanceExtensions()
	extensions, _, err := resource.AvailableExtensions(app.allocator, app.loader)
	if err != nil {
		return err
	}

	for _, ext := range sdlExtensions {
		_, hasExt := extensions[ext]
		if !hasExt {
			return errors.Newf("createinstance: cannot initialize sdl: missing extension %s", ext)
		}
		instanceOptions.ExtensionNames = append(instanceOptions.ExtensionNames, ext)
	}

	if enableValidationLayers {
		instanceOptions.ExtensionNames = append(instanceOptions.ExtensionNames, ext_debugutils2.ExtensionName)
	}

	// Add layers
	layers, _, err := resource.AvailableLayers(app.allocator, app.loader)
	if err != nil {
		return err
	}

	if enableValidationLayers {
		for _, layer := range validationLayers {
			_, hasValidation := layers[layer]
			if !hasValidation {
				return errors.Newf("createInstance: cannot add validation- layer %s not available- install LunarG Vulkan SDK", layer)
			}
			instanceOptions.LayerNames = append(instanceOptions.LayerNames, layer)
		}

		// Add debug messenger
		instanceOptions.Next = app.debugMessengerOptions()
	}

	app.instance, _, err = resource.CreateInstance(app.allocator, app.loader, instanceOptions)
	if err != nil {
		return err
	}

	return nil
}

func (app *HelloTriangleApplication) debugMessengerOptions() *ext_debugutils2.Options {
	return &ext_debugutils2.Options{
		CaptureSeverities: ext_debugutils2.SeverityError | ext_debugutils2.SeverityWarning,
		CaptureTypes:      ext_debugutils2.TypeAll,
		Callback:          app.logDebug,
	}
}

func (app *HelloTriangleApplication) setupDebugMessenger() error {
	if !enableValidationLayers {
		return nil
	}

	var err error
	app.debugMessenger, _, err = ext_debugutils2.CreateMessenger(app.allocator, app.instance, app.debugMessengerOptions())
	if err != nil {
		return err
	}

	return nil
}

func (app *HelloTriangleApplication) createSurface() error {
	surface, _, err := ext_surface_sdl22.CreateSurface(app.allocator, app.instance, &ext_surface_sdl22.CreationOptions{
		Window: app.window,
	})
	if err != nil {
		return err
	}

	app.surface = surface
	return nil
}

func (app *HelloTriangleApplication) pickPhysicalDevice() error {
	physicalDevices, _, err := app.instance.PhysicalDevices(app.allocator)
	if err != nil {
		return err
	}

	for _, device := range physicalDevices {
		if app.isDeviceSuitable(device) {
			app.physicalDevice = device
			break
		}
	}

	if app.physicalDevice == nil {
		return errors.New("failed to find a suitable GPU!")
	}

	return nil
}

func (app *HelloTriangleApplication) createLogicalDevice() error {
	indices, err := app.findQueueFamilies(app.physicalDevice)
	if err != nil {
		return err
	}

	uniqueQueueFamilies := []int{*indices.GraphicsFamily}
	if uniqueQueueFamilies[0] != *indices.PresentFamily {
		uniqueQueueFamilies = append(uniqueQueueFamilies, *indices.PresentFamily)
	}

	var queueFamilyOptions []*resource.QueueFamilyOptions
	queuePriority := float32(1.0)
	for _, queueFamily := range uniqueQueueFamilies {
		queueFamilyOptions = append(queueFamilyOptions, &resource.QueueFamilyOptions{
			QueueFamilyIndex: queueFamily,
			QueuePriorities:  []float32{queuePriority},
		})
	}

	var extensionNames []string
	extensionNames = append(extensionNames, deviceExtensions...)

	var layerNames []string
	if enableValidationLayers {
		layerNames = append(layerNames, validationLayers...)
	}

	app.device, _, err = app.physicalDevice.CreateDevice(app.allocator, &resource.DeviceOptions{
		QueueFamilies:   queueFamilyOptions,
		EnabledFeatures: &core.PhysicalDeviceFeatures{},
		ExtensionNames:  extensionNames,
		LayerNames:      layerNames,
	})
	if err != nil {
		return err
	}

	app.graphicsQueue, err = app.device.GetQueue(*indices.GraphicsFamily, 0)
	if err != nil {
		return err
	}

	app.presentQueue, err = app.device.GetQueue(*indices.PresentFamily, 0)
	return err
}

func (app *HelloTriangleApplication) createSwapchain() error {
	swapchainSupport, err := app.querySwapChainSupport(app.physicalDevice)
	if err != nil {
		return err
	}

	surfaceFormat := app.chooseSwapSurfaceFormat(swapchainSupport.Formats)
	presentMode := app.chooseSwapPresentMode(swapchainSupport.PresentModes)
	extent := app.chooseSwapExtent(swapchainSupport.Capabilities)

	imageCount := swapchainSupport.Capabilities.MinImageCount + 1
	if swapchainSupport.Capabilities.MaxImageCount > 0 && swapchainSupport.Capabilities.MaxImageCount < imageCount {
		imageCount = swapchainSupport.Capabilities.MaxImageCount
	}

	sharingMode := core.SharingExclusive
	var queueFamilyIndices []int

	indices, err := app.findQueueFamilies(app.physicalDevice)
	if err != nil {
		return err
	}

	if *indices.GraphicsFamily != *indices.PresentFamily {
		sharingMode = core.SharingConcurrent
		queueFamilyIndices = append(queueFamilyIndices, *indices.GraphicsFamily, *indices.PresentFamily)
	}

	swapchain, _, err := ext_swapchain2.CreateSwapchain(app.allocator, app.device, &ext_swapchain2.CreationOptions{
		Surface: app.surface,

		MinImageCount:    imageCount,
		ImageFormat:      surfaceFormat.Format,
		ImageColorSpace:  surfaceFormat.ColorSpace,
		ImageExtent:      extent,
		ImageArrayLayers: 1,
		ImageUsage:       core.ImageColorAttachment,

		SharingMode:        sharingMode,
		QueueFamilyIndices: queueFamilyIndices,

		PreTransform:   swapchainSupport.Capabilities.CurrentTransform,
		CompositeAlpha: ext_surface2.Opaque,
		PresentMode:    presentMode,
		Clipped:        true,
	})
	if err != nil {
		return err
	}
	app.swapchainExtent = extent
	app.swapchain = swapchain
	app.swapchainImageFormat = surfaceFormat.Format

	return nil
}

func (app *HelloTriangleApplication) createImageViews() error {
	images, _, err := app.swapchain.Images(app.allocator)
	if err != nil {
		return err
	}
	app.swapchainImages = images

	var imageViews []*resource.ImageView
	for _, image := range images {
		view, _, err := app.device.CreateImageView(app.allocator, &resource.ImageViewOptions{
			ViewType: core.View2D,
			Image:    image,
			Format:   app.swapchainImageFormat,
			Components: core.ComponentMapping{
				R: core.SwizzleIdentity,
				G: core.SwizzleIdentity,
				B: core.SwizzleIdentity,
				A: core.SwizzleIdentity,
			},
			SubresourceRange: core.ImageSubresourceRange{
				AspectMask:     core.AspectColor,
				BaseMipLevel:   0,
				LevelCount:     1,
				BaseArrayLayer: 0,
				LayerCount:     1,
			},
		})
		if err != nil {
			return err
		}

		imageViews = append(imageViews, view)
	}
	app.swapchainImageViews = imageViews

	return nil
}

func (app *HelloTriangleApplication) createRenderPass() error {
	renderPass, _, err := render_pass2.CreateRenderPass(app.allocator, app.device, &render_pass2.RenderPassOptions{
		Attachments: []render_pass2.AttachmentDescription{
			{
				Format:         app.swapchainImageFormat,
				Samples:        core.Samples1,
				LoadOp:         core.LoadOpClear,
				StoreOp:        core.StoreOpStore,
				StencilLoadOp:  core.LoadOpDontCare,
				StencilStoreOp: core.StoreOpDontCare,
				InitialLayout:  core.LayoutUndefined,
				FinalLayout:    core.LayoutPresentSrc,
			},
		},
		SubPasses: []render_pass2.SubPass{
			{
				BindPoint: core.BindGraphics,
				ColorAttachments: []core.AttachmentReference{
					{
						AttachmentIndex: 0,
						Layout:          core.LayoutColorAttachmentOptimal,
					},
				},
			},
		},
		SubPassDependencies: []render_pass2.SubPassDependency{
			{
				SrcSubPassIndex: render_pass2.SubpassExternal,
				DstSubPassIndex: 0,

				SrcStageMask: core.PipelineStageColorAttachmentOutput,
				SrcAccess:    0,

				DstStageMask: core.PipelineStageColorAttachmentOutput,
				DstAccess:    core.AccessColorAttachmentWrite,
			},
		},
	})
	if err != nil {
		return err
	}

	app.renderPass = renderPass

	return nil
}

func bytesToBytecode(b []byte) []uint32 {
	byteCode := make([]uint32, len(b)/4)
	for i := 0; i < len(byteCode); i++ {
		byteIndex := i * 4
		byteCode[i] = 0
		byteCode[i] |= uint32(b[byteIndex])
		byteCode[i] |= uint32(b[byteIndex+1]) << 8
		byteCode[i] |= uint32(b[byteIndex+2]) << 16
		byteCode[i] |= uint32(b[byteIndex+3]) << 24
	}

	return byteCode
}

func (app *HelloTriangleApplication) createGraphicsPipeline() error {
	// Load vertex shader
	vertShaderBytes, err := shaders.ReadFile("shaders/vert.spv")
	if err != nil {
		return err
	}

	vertShader, _, err := app.device.CreateShaderModule(app.allocator, &resource.ShaderModuleOptions{
		SpirVByteCode: bytesToBytecode(vertShaderBytes),
	})
	if err != nil {
		return err
	}
	defer vertShader.Destroy()

	// Load fragment shader
	fragShaderBytes, err := shaders.ReadFile("shaders/frag.spv")
	if err != nil {
		return err
	}

	fragShader, _, err := app.device.CreateShaderModule(app.allocator, &resource.ShaderModuleOptions{
		SpirVByteCode: bytesToBytecode(fragShaderBytes),
	})
	if err != nil {
		return err
	}
	defer fragShader.Destroy()

	vertexInput := &pipeline2.VertexInputOptions{
		VertexBindingDescriptions:   getVertexBindingDescription(),
		VertexAttributeDescriptions: getVertexAttributeDescriptions(),
	}

	inputAssembly := &pipeline2.InputAssemblyOptions{
		Topology:               core.TopologyTriangleList,
		EnablePrimitiveRestart: false,
	}

	vertStage := &pipeline2.ShaderStage{
		Stage:  core.StageVertex,
		Shader: vertShader,
		Name:   "main",
	}

	fragStage := &pipeline2.ShaderStage{
		Stage:  core.StageFragment,
		Shader: fragShader,
		Name:   "main",
	}

	viewport := &pipeline2.ViewportOptions{
		Viewports: []core.Viewport{
			{
				X:        0,
				Y:        0,
				Width:    float32(app.swapchainExtent.Width),
				Height:   float32(app.swapchainExtent.Height),
				MinDepth: 0,
				MaxDepth: 1,
			},
		},
		Scissors: []core.Rect2D{
			{
				Offset: core.Offset2D{X: 0, Y: 0},
				Extent: app.swapchainExtent,
			},
		},
	}

	rasterization := &pipeline2.RasterizationOptions{
		DepthClamp:        false,
		RasterizerDiscard: false,

		PolygonMode: pipeline2.ModeFill,
		CullMode:    core.CullBack,
		FrontFace:   core.Clockwise,

		DepthBias: false,

		LineWidth: 1.0,
	}

	multisample := &pipeline2.MultisampleOptions{
		SampleShading:        false,
		RasterizationSamples: core.Samples1,
		MinSampleShading:     1.0,
	}

	colorBlend := &pipeline2.ColorBlendOptions{
		LogicOpEnabled: false,
		LogicOp:        core.LogicOpCopy,

		BlendConstants: [4]float32{0, 0, 0, 0},
		Attachments: []pipeline2.ColorBlendAttachment{
			{
				BlendEnabled: false,
				WriteMask:    core.ComponentRed | core.ComponentGreen | core.ComponentBlue | core.ComponentAlpha,
			},
		},
	}

	app.pipelineLayout, _, err = pipeline2.CreatePipelineLayout(app.allocator, app.device, &pipeline2.PipelineLayoutOptions{})
	if err != nil {
		return err
	}

	pipelines, _, err := pipeline2.CreateGraphicsPipelines(app.allocator, app.device, []*pipeline2.Options{
		{
			ShaderStages: []*pipeline2.ShaderStage{
				vertStage,
				fragStage,
			},
			VertexInput:       vertexInput,
			InputAssembly:     inputAssembly,
			Viewport:          viewport,
			Rasterization:     rasterization,
			Multisample:       multisample,
			ColorBlend:        colorBlend,
			Layout:            app.pipelineLayout,
			RenderPass:        app.renderPass,
			SubPass:           0,
			BasePipelineIndex: -1,
		},
	})
	if err != nil {
		return err
	}
	app.graphicsPipeline = pipelines[0]

	return nil
}

func (app *HelloTriangleApplication) createFramebuffers() error {
	for _, imageView := range app.swapchainImageViews {
		framebuffer, _, err := render_pass2.CreateFrameBuffer(app.allocator, app.device, &render_pass2.FramebufferOptions{
			RenderPass: app.renderPass,
			Layers:     1,
			Attachments: []*resource.ImageView{
				imageView,
			},
			Width:  app.swapchainExtent.Width,
			Height: app.swapchainExtent.Height,
		})
		if err != nil {
			return err
		}

		app.swapchainFramebuffers = append(app.swapchainFramebuffers, framebuffer)
	}

	return nil
}

func (app *HelloTriangleApplication) createCommandPool() error {
	indices, err := app.findQueueFamilies(app.physicalDevice)
	if err != nil {
		return err
	}

	pool, _, err := commands2.CreateCommandPool(app.allocator, app.device, &commands2.CommandPoolOptions{
		GraphicsQueueFamily: indices.GraphicsFamily,
	})

	if err != nil {
		return err
	}
	app.commandPool = pool

	return nil
}

func (app *HelloTriangleApplication) createCommandBuffers() error {

	buffers, _, err := commands2.CreateCommandBuffers(app.allocator, app.device, &commands2.CommandBufferOptions{
		Level:       core.LevelPrimary,
		BufferCount: len(app.swapchainImages),
		CommandPool: app.commandPool,
	})
	if err != nil {
		return err
	}
	app.commandBuffers = buffers

	for bufferIdx, buffer := range buffers {
		_, err = buffer.Begin(app.allocator, &commands2.BeginOptions{})
		if err != nil {
			return err
		}

		err = buffer.CmdBeginRenderPass(app.allocator, commands2.ContentsInline,
			&commands2.RenderPassBeginOptions{
				RenderPass:  app.renderPass,
				Framebuffer: app.swapchainFramebuffers[bufferIdx],
				RenderArea: core.Rect2D{
					Offset: core.Offset2D{X: 0, Y: 0},
					Extent: app.swapchainExtent,
				},
				ClearValues: []commands2.ClearValue{
					commands2.ClearValueFloat{0, 0, 0, 1},
				},
			})
		if err != nil {
			return err
		}

		buffer.CmdBindPipeline(core.BindGraphics, app.graphicsPipeline)
		buffer.CmdDraw(3, 1, 0, 0)
		buffer.CmdEndRenderPass()

		_, err = buffer.End()
		if err != nil {
			return err
		}
	}

	return nil
}

func (app *HelloTriangleApplication) createSyncObjects() error {
	for i := 0; i < MaxFramesInFlight; i++ {
		semaphore, _, err := app.device.CreateSemaphore(app.allocator, &resource.SemaphoreOptions{})
		if err != nil {
			return err
		}

		app.imageAvailableSemaphore = append(app.imageAvailableSemaphore, semaphore)

		semaphore, _, err = app.device.CreateSemaphore(app.allocator, &resource.SemaphoreOptions{})
		if err != nil {
			return err
		}

		app.renderFinishedSemaphore = append(app.renderFinishedSemaphore, semaphore)

		fence, _, err := app.device.CreateFence(app.allocator, &resource.FenceOptions{
			Flags: resource.FenceSignaled,
		})
		if err != nil {
			return err
		}

		app.inFlightFence = append(app.inFlightFence, fence)
	}

	for i := 0; i < len(app.swapchainImages); i++ {
		app.imagesInFlight = append(app.imagesInFlight, nil)
	}

	return nil
}

func (app *HelloTriangleApplication) drawFrame() error {
	fences := []*resource.Fence{app.inFlightFence[app.currentFrame]}

	_, err := app.device.WaitForFences(app.allocator, true, core.NoTimeout, fences)
	if err != nil {
		return err
	}

	imageIndex, res, err := app.swapchain.AcquireNextImage(core.NoTimeout, app.imageAvailableSemaphore[app.currentFrame], nil)
	if res == loader.VKErrorOutOfDate {
		return app.recreateSwapChain()
	} else if err != nil {
		return err
	}

	if app.imagesInFlight[imageIndex] != nil {
		_, err := app.device.WaitForFences(app.allocator, true, core.NoTimeout, []*resource.Fence{app.imagesInFlight[imageIndex]})
		if err != nil {
			return err
		}
	}
	app.imagesInFlight[imageIndex] = app.inFlightFence[app.currentFrame]

	_, err = app.device.ResetFences(app.allocator, fences)
	if err != nil {
		return err
	}

	_, err = commands2.SubmitToQueue(app.allocator, app.graphicsQueue, app.inFlightFence[app.currentFrame], []*commands2.SubmitOptions{
		{
			WaitSemaphores:   []*resource.Semaphore{app.imageAvailableSemaphore[app.currentFrame]},
			WaitDstStages:    []core.PipelineStages{core.PipelineStageColorAttachmentOutput},
			CommandBuffers:   []*commands2.CommandBuffer{app.commandBuffers[imageIndex]},
			SignalSemaphores: []*resource.Semaphore{app.renderFinishedSemaphore[app.currentFrame]},
		},
	})
	if err != nil {
		return err
	}

	results, _, err := app.swapchain.PresentToQueue(app.allocator, app.presentQueue, &ext_swapchain2.PresentOptions{
		WaitSemaphores: []*resource.Semaphore{app.renderFinishedSemaphore[app.currentFrame]},
		Swapchains:     []*ext_swapchain2.Swapchain{app.swapchain},
		ImageIndices:   []int{imageIndex},
	})
	if results[0] == loader.VKErrorOutOfDate || results[0] == loader.VKSuboptimal {
		return app.recreateSwapChain()
	} else if err != nil {
		return err
	}

	app.currentFrame = (app.currentFrame + 1) % MaxFramesInFlight

	return nil
}

func (app *HelloTriangleApplication) chooseSwapSurfaceFormat(availableFormats []ext_surface2.Format) ext_surface2.Format {
	for _, format := range availableFormats {
		if format.Format == core.FormatB8G8R8A8SRGB && format.ColorSpace == ext_surface2.SRGBNonlinear {
			return format
		}
	}

	return availableFormats[0]
}

func (app *HelloTriangleApplication) chooseSwapPresentMode(availablePresentModes []ext_surface2.PresentMode) ext_surface2.PresentMode {
	for _, presentMode := range availablePresentModes {
		if presentMode == ext_surface2.Mailbox {
			return presentMode
		}
	}

	return ext_surface2.FIFO
}

func (app *HelloTriangleApplication) chooseSwapExtent(capabilities *ext_surface2.Capabilities) core.Extent2D {
	if capabilities.CurrentExtent.Width != (^uint32(0)) {
		return capabilities.CurrentExtent
	}

	widthInt, heightInt := app.window.VulkanGetDrawableSize()
	width := uint32(widthInt)
	height := uint32(heightInt)

	if width < capabilities.MinImageExtent.Width {
		width = capabilities.MinImageExtent.Width
	}
	if width > capabilities.MaxImageExtent.Width {
		width = capabilities.MaxImageExtent.Width
	}
	if height < capabilities.MinImageExtent.Height {
		height = capabilities.MinImageExtent.Height
	}
	if height > capabilities.MaxImageExtent.Height {
		height = capabilities.MaxImageExtent.Height
	}

	return core.Extent2D{Width: width, Height: height}
}

func (app *HelloTriangleApplication) querySwapChainSupport(device *resource.PhysicalDevice) (SwapChainSupportDetails, error) {
	var details SwapChainSupportDetails
	var err error

	details.Capabilities, _, err = app.surface.Capabilities(app.allocator, device)
	if err != nil {
		return details, err
	}

	details.Formats, _, err = app.surface.Formats(app.allocator, device)
	if err != nil {
		return details, err
	}

	details.PresentModes, _, err = app.surface.PresentModes(app.allocator, device)
	return details, err
}

func (app *HelloTriangleApplication) isDeviceSuitable(device *resource.PhysicalDevice) bool {
	indices, err := app.findQueueFamilies(device)
	if err != nil {
		return false
	}

	extensionsSupported := app.checkDeviceExtensionSupport(device)

	var swapChainAdequate bool
	if extensionsSupported {
		swapChainSupport, err := app.querySwapChainSupport(device)
		if err != nil {
			return false
		}

		swapChainAdequate = len(swapChainSupport.Formats) > 0 && len(swapChainSupport.PresentModes) > 0
	}

	return indices.IsComplete() && extensionsSupported && swapChainAdequate
}

func (app *HelloTriangleApplication) checkDeviceExtensionSupport(device *resource.PhysicalDevice) bool {
	extensions, _, err := device.AvailableExtensions(app.allocator)
	if err != nil {
		return false
	}

	for _, extension := range deviceExtensions {
		_, hasExtension := extensions[extension]
		if !hasExtension {
			return false
		}
	}

	return true
}

func (app *HelloTriangleApplication) findQueueFamilies(device *resource.PhysicalDevice) (QueueFamilyIndices, error) {
	indices := QueueFamilyIndices{}
	queueFamilies, err := device.QueueFamilyProperties(app.allocator)
	if err != nil {
		return indices, err
	}

	for queueFamilyIdx, queueFamily := range queueFamilies {
		if (queueFamily.Flags & core.Graphics) != 0 {
			indices.GraphicsFamily = new(int)
			*indices.GraphicsFamily = queueFamilyIdx
		}

		supported, _, err := app.surface.SupportsDevice(device, queueFamilyIdx)
		if err != nil {
			return indices, err
		}

		if supported {
			indices.PresentFamily = new(int)
			*indices.PresentFamily = queueFamilyIdx
		}

		if indices.IsComplete() {
			break
		}
	}

	return indices, nil
}

func (app *HelloTriangleApplication) logDebug(msgType ext_debugutils2.MessageType, severity ext_debugutils2.MessageSeverity, data *ext_debugutils2.CallbackData) bool {
	log.Printf("[%s %s] - %s", severity, msgType, data.Message)
	return false
}

func main() {
	defAlloc := &cgoalloc.DefaultAllocator{}
	lowTier, err := cgoalloc.CreateFixedBlockAllocator(defAlloc, 64*1024, 64, 8)
	if err != nil {
		log.Fatalln(err)
	}

	highTier, err := cgoalloc.CreateFixedBlockAllocator(defAlloc, 4096*1024, 4096, 8)
	if err != nil {
		log.Fatalln(err)
	}

	alloc := cgoalloc.CreateFallbackAllocator(highTier, defAlloc)
	alloc = cgoalloc.CreateFallbackAllocator(lowTier, alloc)

	app := &HelloTriangleApplication{
		allocator: alloc,
	}

	err = app.Run()
	if err != nil {
		log.Fatalln(err)
	}
}
