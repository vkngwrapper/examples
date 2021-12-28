package utils

import (
	"bytes"
	"encoding/binary"
	"github.com/CannibalVox/VKng/core"
	"github.com/CannibalVox/VKng/core/common"
	"github.com/CannibalVox/VKng/extensions/khr_surface"
	"github.com/CannibalVox/VKng/extensions/khr_surface_sdl2"
	"github.com/CannibalVox/VKng/extensions/khr_swapchain"
	"github.com/cockroachdb/errors"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/veandco/go-sdl2/sdl"
	"log"
	"unsafe"
)

type LayerProperties struct {
	Properties         *common.LayerProperties
	InstanceExtensions []*common.ExtensionProperties
	DeviceExtensions   []*common.ExtensionProperties
}

type SwapchainBuffer struct {
	Image core.Image
	View  core.ImageView
}

type SampleInfo struct {
	Loader           core.Loader1_0
	Window           *sdl.Window
	Surface          khr_surface.Surface
	Prepared         bool
	UseStagingBuffer bool
	SaveImages       bool

	InstanceLayerNames          []string
	InstanceExtensionNames      []string
	InstanceLayerProperties     []*LayerProperties
	InstanceExtensionProperties []*common.ExtensionProperties
	Instance                    core.Instance

	DeviceExtensionNames      []string
	DeviceExtensionProperties []*common.ExtensionProperties
	Gpus                      []core.PhysicalDevice
	Device                    core.Device
	GraphicsQueue             core.Queue
	PresentQueue              core.Queue
	GraphicsQueueFamilyIndex  int
	PresentQueueFamilyIndex   int
	GpuProps                  *common.PhysicalDeviceProperties
	QueueProps                []*common.QueueFamily
	MemoryProperties          *core.PhysicalDeviceMemoryProperties

	Framebuffer   []core.Framebuffer
	Width, Height int
	Format        common.DataFormat

	SwapchainImageCount    int
	Swapchain              khr_swapchain.Swapchain
	Buffers                []SwapchainBuffer
	ImageAcquiredSemaphore core.Semaphore

	CmdPool core.CommandPool

	Depth struct {
		Format common.DataFormat
		Image  core.Image
		Mem    core.DeviceMemory
		View   core.ImageView
	}

	Textures []*TextureObject

	UniformData struct {
		Buf        core.Buffer
		Mem        core.DeviceMemory
		BufferInfo core.DescriptorBufferInfo
	}

	TextureData struct {
		ImageInfo core.DescriptorImageInfo
	}

	VertexBuffer struct {
		Buf        core.Buffer
		Mem        core.DeviceMemory
		BufferInfo core.DescriptorBufferInfo
	}

	VertexBinding    core.VertexBindingDescription
	VertexAttributes []core.VertexAttributeDescription

	Projection mgl32.Mat4
	View       mgl32.Mat4
	Model      mgl32.Mat4
	Clip       mgl32.Mat4
	MVP        mgl32.Mat4

	Cmd            core.CommandBuffer // BUffer for initialization commands
	PipelineLayout core.PipelineLayout
	DescLayout     []core.DescriptorSetLayout
	PipelineCache  core.PipelineCache
	RenderPass     core.RenderPass
	Pipeline       core.Pipeline

	ShaderStages []*core.ShaderStage

	DescPool core.DescriptorPool
	DescSet  []core.DescriptorSet

	//PFN_vkCreateDebugReportCallbackEXT dbgCreateDebugReportCallback;
	//PFN_vkDestroyDebugReportCallbackEXT dbgDestroyDebugReportCallback;
	//PFN_vkDebugReportMessageEXT dbgBreakCallback;
	//std::vector<VkDebugReportCallbackEXT> debug_report_callbacks;

	CurrentBuffer    int
	QueueFamilyCount int

	Viewport common.Viewport
	Scissor  common.Rect2D
}

func (i *SampleInfo) InitWindowSize(defaultWidth, defaultHeight int) error {
	i.Width = defaultWidth
	i.Height = defaultHeight
	return nil
}

func (i *SampleInfo) InitWindow() error {
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		return err
	}

	window, err := sdl.CreateWindow("Sample", 100, 100, int32(i.Width), int32(i.Height), sdl.WINDOW_SHOWN|sdl.WINDOW_VULKAN|sdl.WINDOW_RESIZABLE)
	if err != nil {
		return err
	}
	i.Window = window
	return nil
}

func (i *SampleInfo) InitGlobalLayerProperties() error {
	layers, _, err := i.Loader.AvailableLayers()
	if err != nil {
		return err
	}
	for _, properties := range layers {
		props := &LayerProperties{
			Properties: properties,
		}
		err = i.InitGlobalExtensionProperties(props)
		if err != nil {
			return err
		}
		i.InstanceLayerProperties = append(i.InstanceLayerProperties, props)
	}

	return nil
}

func (i *SampleInfo) InitGlobalExtensionProperties(layerProps *LayerProperties) error {
	instanceExtensions, _, err := i.Loader.AvailableExtensionsForLayer(layerProps.Properties.LayerName)
	if err != nil {
		return err
	}

	for _, props := range instanceExtensions {
		layerProps.InstanceExtensions = append(layerProps.InstanceExtensions, props)
	}

	return nil
}

func (i *SampleInfo) InitInstanceExtensionNames() error {
	i.InstanceExtensionNames = i.Window.VulkanGetInstanceExtensions()
	return nil
}

func (i *SampleInfo) InitInstance(appShortName string, next common.Options) error {
	var err error
	i.Instance, _, err = i.Loader.CreateInstance(&core.InstanceOptions{
		ApplicationName:    appShortName,
		ApplicationVersion: common.CreateVersion(0, 0, 1),
		EngineName:         appShortName,
		EngineVersion:      common.CreateVersion(0, 0, 1),
		VulkanVersion:      common.Vulkan1_0,
		ExtensionNames:     i.InstanceExtensionNames,
		LayerNames:         i.InstanceLayerNames,
		HaveNext: common.HaveNext{
			Next: next,
		},
	})
	return err
}

func (i *SampleInfo) InitDeviceExtensionNames() error {
	i.DeviceExtensionNames = []string{khr_swapchain.ExtensionName}
	return nil
}

func (i *SampleInfo) InitEnumerateDevice() error {
	var err error
	i.Gpus, _, err = i.Instance.PhysicalDevices()
	if err != nil {
		return err
	}

	i.QueueProps = i.Gpus[0].QueueFamilyProperties()
	i.QueueFamilyCount = len(i.QueueProps)

	i.MemoryProperties = i.Gpus[0].MemoryProperties()
	i.GpuProps = i.Gpus[0].Properties()

	for _, layerProps := range i.InstanceLayerProperties {
		err = i.InitDeviceExtensionProperties(layerProps)
		if err != nil {
			return err
		}
	}

	return nil
}

func (i *SampleInfo) InitDeviceExtensionProperties(layerProps *LayerProperties) error {
	deviceExtensions, _, err := i.Gpus[0].AvailableExtensionsForLayer(layerProps.Properties.LayerName)
	if err != nil {
		return err
	}

	for _, deviceExtension := range deviceExtensions {
		layerProps.DeviceExtensions = append(layerProps.DeviceExtensions, deviceExtension)
	}

	return nil
}

func (i *SampleInfo) InitSwapchainExtension() error {
	// Construct the surface
	surfaceLoader := khr_surface_sdl2.CreateLoaderFromInstance(i.Instance)

	var err error
	i.Surface, _, err = surfaceLoader.CreateSurface(i.Instance, i.Window)
	if err != nil {
		return err
	}

	// Iterate over each queue to learn whether it supports presenting:
	var presentSupport []bool
	for queueIndex := range i.QueueProps {
		support, _, err := i.Surface.SupportsDevice(i.Gpus[0], queueIndex)
		if err != nil {
			return err
		}
		presentSupport = append(presentSupport, support)
	}

	// Search for a graphics and a present queue in the array of queue
	// families, try to find one that supports both
	i.GraphicsQueueFamilyIndex = -1
	i.PresentQueueFamilyIndex = -1
	for queueIndex, queueFamily := range i.QueueProps {
		if (queueFamily.Flags & common.QueueGraphics) != 0 {
			if i.GraphicsQueueFamilyIndex < 0 {
				i.GraphicsQueueFamilyIndex = queueIndex
			}

			if presentSupport[queueIndex] {
				i.GraphicsQueueFamilyIndex = queueIndex
				i.PresentQueueFamilyIndex = queueIndex
				break
			}
		}
	}

	if i.PresentQueueFamilyIndex < 0 {
		// If didn't find a queue that supports both graphics and present, then
		// find a separate present queue.
		for queueIndex := range i.QueueProps {
			if presentSupport[queueIndex] {
				i.PresentQueueFamilyIndex = queueIndex
				break
			}
		}
	}

	// Generate error if could not find queues that support graphics
	// and present
	if i.GraphicsQueueFamilyIndex < 0 || i.PresentQueueFamilyIndex < 0 {
		return errors.New("could not find a queue for both graphics and present")
	}

	// Get the list of VkFormats that are supported:
	formats, _, err := i.Surface.Formats(i.Gpus[0])
	if err != nil {
		return err
	}

	// If the device supports our preferred surface format, use it.
	// Otherwise, use whatever the device's first reported surface
	// format is.
	i.Format = formats[0].Format
	for _, format := range formats {
		if format.Format == PreferredSurfaceFormat {
			i.Format = PreferredSurfaceFormat
			break
		}
	}

	return nil
}

func (i *SampleInfo) InitDevice() error {
	var err error
	i.Device, _, err = i.Loader.CreateDevice(i.Gpus[0], &core.DeviceOptions{
		QueueFamilies: []*core.QueueFamilyOptions{
			{
				QueueFamilyIndex: i.GraphicsQueueFamilyIndex,
				QueuePriorities:  []float32{0.0},
			},
		},
		ExtensionNames: i.DeviceExtensionNames,
	})
	return err
}

func (i *SampleInfo) InitCommandPool() error {
	var err error
	i.CmdPool, _, err = i.Loader.CreateCommandPool(i.Device, &core.CommandPoolOptions{
		GraphicsQueueFamily: &i.GraphicsQueueFamilyIndex,
		Flags:               core.CommandPoolResetBuffer,
	})
	return err
}

func (i *SampleInfo) InitCommandBuffer() error {
	buffers, _, err := i.CmdPool.AllocateCommandBuffers(&core.CommandBufferOptions{
		Level:       common.LevelPrimary,
		BufferCount: 1,
	})
	if err != nil {
		return err
	}
	i.Cmd = buffers[0]
	return nil
}

func (i *SampleInfo) ExecuteBeginCommandBuffer() error {
	_, err := i.Cmd.Begin(&core.BeginOptions{})
	return err
}

func (i *SampleInfo) ExecuteEndCommandBuffer() error {
	_, err := i.Cmd.End()
	return err
}

func (i *SampleInfo) InitDeviceQueue() error {
	i.GraphicsQueue = i.Device.GetQueue(i.GraphicsQueueFamilyIndex, 0)

	if i.PresentQueueFamilyIndex == i.GraphicsQueueFamilyIndex {
		i.PresentQueue = i.GraphicsQueue
		return nil
	}

	i.PresentQueue = i.Device.GetQueue(i.PresentQueueFamilyIndex, 0)
	return nil
}

func (i *SampleInfo) InitSwapchain(usage common.ImageUsages) error {
	surfaceCaps, _, err := i.Surface.Capabilities(i.Gpus[0])
	if err != nil {
		return err
	}

	var swapchainExtent common.Extent2D
	if surfaceCaps.CurrentExtent.Width < 0 {
		// If the surface size is undefined, the size is set to
		// the size of the images requested.
		swapchainExtent.Width = i.Width
		swapchainExtent.Height = i.Height
		if swapchainExtent.Width < surfaceCaps.MinImageExtent.Width {
			swapchainExtent.Width = surfaceCaps.MinImageExtent.Width
		} else if swapchainExtent.Width > surfaceCaps.MaxImageExtent.Width {
			swapchainExtent.Width = surfaceCaps.MaxImageExtent.Width
		}

		if swapchainExtent.Height < surfaceCaps.MinImageExtent.Height {
			swapchainExtent.Height = surfaceCaps.MinImageExtent.Height
		} else if swapchainExtent.Height > surfaceCaps.MaxImageExtent.Height {
			swapchainExtent.Height = surfaceCaps.MaxImageExtent.Height
		}
	} else {
		// If the surface size is defined, the swap chain size must match
		swapchainExtent = surfaceCaps.CurrentExtent
	}

	// The FIFO present mode is guaranteed by the spec to be supported
	// Also note that current Android driver only supports FIFO
	presentMode := khr_surface.PresentFIFO

	// Determine the number of VkImage's to use in the swap chain.
	// We need to acquire only 1 presentable image at a time.
	// Asking for minImageCount images ensures that we can acquire
	// 1 presentable image as long as we present it before attempting
	// to acquire another.
	desiredNumberOfSwapChainImages := surfaceCaps.MinImageCount

	var preTransform khr_surface.SurfaceTransforms
	if (surfaceCaps.SupportedTransforms & khr_surface.TransformIdentity) != 0 {
		preTransform = khr_surface.TransformIdentity
	} else {
		preTransform = surfaceCaps.CurrentTransform
	}

	// Find a supported composite alpha mode - one of these is guaranteed to be set
	compositeAlpha := khr_surface.AlphaModeOpaque
	compositeAlphaFlags := [4]khr_surface.CompositeAlphaModes{
		khr_surface.AlphaModeOpaque,
		khr_surface.AlphaModePreMultiplied,
		khr_surface.AlphaModePostMultiplied,
		khr_surface.AlphaModeInherit,
	}

	for i := 0; i < len(compositeAlphaFlags); i++ {
		if (surfaceCaps.SupportedCompositeAlpha & compositeAlphaFlags[i]) != 0 {
			compositeAlpha = compositeAlphaFlags[i]
			break
		}
	}

	swapchainLoader := khr_swapchain.CreateLoaderFromDevice(i.Device)
	swapchainOptions := &khr_swapchain.CreationOptions{
		Surface:          i.Surface,
		MinImageCount:    desiredNumberOfSwapChainImages,
		ImageFormat:      i.Format,
		ImageExtent:      swapchainExtent,
		PreTransform:     preTransform,
		CompositeAlpha:   compositeAlpha,
		ImageArrayLayers: 1,
		PresentMode:      presentMode,
		Clipped:          true,
		ImageColorSpace:  khr_surface.ColorSpaceSRGBNonlinear,
		ImageUsage:       usage,
		SharingMode:      common.SharingExclusive,
	}

	if i.GraphicsQueueFamilyIndex != i.PresentQueueFamilyIndex {
		// If the graphics and present queues are from different queue families,
		// we either have to explicitly transfer ownership of images between the
		// queues, or we have to create the swapchain with imageSharingMode
		// as VK_SHARING_MODE_CONCURRENT
		swapchainOptions.SharingMode = common.SharingConcurrent
		swapchainOptions.QueueFamilyIndices = []int{
			i.GraphicsQueueFamilyIndex,
			i.PresentQueueFamilyIndex,
		}
	}

	i.Swapchain, _, err = swapchainLoader.CreateSwapchain(i.Device, swapchainOptions)
	if err != nil {
		return err
	}

	images, _, err := i.Swapchain.Images()
	if err != nil {
		return err
	}
	i.SwapchainImageCount = len(images)

	for _, image := range images {
		view, _, err := i.Loader.CreateImageView(i.Device, &core.ImageViewOptions{
			Image:    image,
			ViewType: common.ViewType2D,
			Format:   i.Format,
			Components: common.ComponentMapping{
				R: common.SwizzleRed,
				G: common.SwizzleGreen,
				B: common.SwizzleBlue,
				A: common.SwizzleAlpha,
			},
			SubresourceRange: common.ImageSubresourceRange{
				AspectMask:     common.AspectColor,
				BaseMipLevel:   0,
				LevelCount:     1,
				BaseArrayLayer: 0,
				LayerCount:     1,
			},
		})
		if err != nil {
			return err
		}

		i.Buffers = append(i.Buffers, SwapchainBuffer{
			Image: image,
			View:  view,
		})
	}

	i.CurrentBuffer = 0
	return nil
}

func (i *SampleInfo) InitDepthBuffer() error {
	if i.Depth.Format == common.FormatUndefined {
		i.Depth.Format = common.FormatD16UnsignedNormalized
	}
	depthFormat := i.Depth.Format

	props := i.Gpus[0].FormatProperties(depthFormat)

	imageOptions := &core.ImageOptions{
		ImageType: common.ImageType2D,
		Format:    depthFormat,
		Extent: common.Extent3D{
			Width:  i.Width,
			Height: i.Height,
			Depth:  1,
		},
		MipLevels:     1,
		ArrayLayers:   1,
		Samples:       NumSamples,
		InitialLayout: common.LayoutUndefined,
		SharingMode:   common.SharingExclusive,
		Usage:         common.ImageUsageDepthStencilAttachment,
	}
	if (props.LinearTilingFeatures & common.FormatFeatureDepthStencilAttachment) != 0 {
		imageOptions.Tiling = common.ImageTilingLinear
	} else if (props.OptimalTilingFeatures & common.FormatFeatureDepthStencilAttachment) != 0 {
		imageOptions.Tiling = common.ImageTilingOptimal
	} else {
		return errors.Newf("depth format %s unsupported", depthFormat)
	}

	var err error
	i.Depth.Image, _, err = i.Loader.CreateImage(i.Device, imageOptions)
	if err != nil {
		return err
	}

	imageMemoryReqs := i.Depth.Image.MemoryRequirements()
	typeIndex, err := i.MemoryTypeFromProperties(imageMemoryReqs.MemoryType, core.MemoryDeviceLocal)
	if err != nil {
		return err
	}

	i.Depth.Mem, _, err = i.Device.AllocateMemory(&core.DeviceMemoryOptions{
		AllocationSize:  imageMemoryReqs.Size,
		MemoryTypeIndex: typeIndex,
	})
	if err != nil {
		return err
	}

	_, err = i.Depth.Image.BindImageMemory(i.Depth.Mem, 0)
	if err != nil {
		return err
	}

	i.Depth.View, _, err = i.Loader.CreateImageView(i.Device, &core.ImageViewOptions{
		Image:  i.Depth.Image,
		Format: depthFormat,
		Components: common.ComponentMapping{
			A: common.SwizzleAlpha,
			R: common.SwizzleRed,
			G: common.SwizzleGreen,
			B: common.SwizzleBlue,
		},
		SubresourceRange: common.ImageSubresourceRange{
			AspectMask:     common.AspectDepth,
			BaseMipLevel:   0,
			LevelCount:     1,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
		ViewType: common.ViewType2D,
	})
	return err
}

func (i *SampleInfo) MemoryTypeFromProperties(memoryType uint32, flags core.MemoryPropertyFlags) (int, error) {
	for typeIndex, memType := range i.MemoryProperties.MemoryTypes {
		if (memoryType & 1) != 0 {
			// Type is available, does it match user properties?
			if (memType.Properties & flags) == flags {
				return typeIndex, nil
			}
		}
		memoryType >>= 1
	}

	return 0, errors.Newf("could not find a memory type matching type request %x with flags %s", memoryType, flags)
}

func (i *SampleInfo) InitUniformBuffer() error {
	fov := mgl32.DegToRad(45)
	if i.Width > i.Height {
		fov *= float32(i.Height) / float32(i.Width)
	}

	i.Projection = mgl32.Perspective(fov, float32(i.Width)/float32(i.Height), 0.1, 100)
	i.View = mgl32.LookAt(-5, 3, -10, 0, 0, 0, 0, -1, 0)
	i.Model = mgl32.Ident4()
	i.Clip = mgl32.Mat4{1, 0, 0, 0, 0, -1, 0, 0, 0, 0, 0.5, 0, 0, 0, 0.5, 1}

	i.MVP = i.Clip.Mul4(i.Projection).Mul4(i.View).Mul4(i.Model)

	var err error
	i.UniformData.Buf, _, err = i.Loader.CreateBuffer(i.Device, &core.BufferOptions{
		Usage:       common.UsageUniformBuffer,
		BufferSize:  int(unsafe.Sizeof(i.MVP)),
		SharingMode: common.SharingExclusive,
	})
	if err != nil {
		return err
	}

	memReqs := i.UniformData.Buf.MemoryRequirements()
	memoryTypeIndex, err := i.MemoryTypeFromProperties(memReqs.MemoryType, core.MemoryHostVisible|core.MemoryHostCoherent)
	if err != nil {
		return err
	}

	i.UniformData.Mem, _, err = i.Device.AllocateMemory(&core.DeviceMemoryOptions{
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memoryTypeIndex,
	})
	if err != nil {
		return err
	}

	memPtr, _, err := i.UniformData.Mem.MapMemory(0, memReqs.Size, 0)
	if err != nil {
		return err
	}

	dataBuffer := unsafe.Slice((*byte)(memPtr), memReqs.Size)

	buf := &bytes.Buffer{}
	err = binary.Write(buf, common.ByteOrder, i.MVP)
	if err != nil {
		i.UniformData.Mem.UnmapMemory()
		return err
	}

	copy(dataBuffer, buf.Bytes())

	i.UniformData.Mem.UnmapMemory()
	if err != nil {
		return err
	}

	_, err = i.UniformData.Buf.BindBufferMemory(i.UniformData.Mem, 0)
	if err != nil {
		return err
	}

	i.UniformData.BufferInfo.Buffer = i.UniformData.Buf
	i.UniformData.BufferInfo.Offset = 0
	i.UniformData.BufferInfo.Range = int(unsafe.Sizeof(i.MVP))
	return nil
}

func (i *SampleInfo) InitDescriptorAndPipelineLayouts(useTexture bool) error {
	layoutBindings := []*core.DescriptorLayoutBinding{
		{
			Binding:         0,
			DescriptorType:  common.DescriptorUniformBuffer,
			DescriptorCount: 1,
			StageFlags:      common.StageVertex,
		},
	}
	if useTexture {
		layoutBindings = append(layoutBindings, &core.DescriptorLayoutBinding{
			Binding:         1,
			DescriptorType:  common.DescriptorCombinedImageSampler,
			DescriptorCount: 1,
			StageFlags:      common.StageFragment,
		})
	}

	layout, _, err := i.Loader.CreateDescriptorSetLayout(i.Device, &core.DescriptorSetLayoutOptions{
		Bindings: layoutBindings,
	})
	if err != nil {
		return err
	}

	i.DescLayout = []core.DescriptorSetLayout{layout}
	i.PipelineLayout, _, err = i.Loader.CreatePipelineLayout(i.Device, &core.PipelineLayoutOptions{
		SetLayouts: []core.DescriptorSetLayout{layout},
	})
	return err
}

func (i *SampleInfo) InitRenderPass(depthPresent, clear bool, finalLayout, initialLayout common.ImageLayout) error {
	attachments := []core.AttachmentDescription{
		{
			Format:         i.Format,
			Samples:        NumSamples,
			LoadOp:         common.LoadOpClear,
			StoreOp:        common.StoreOpStore,
			StencilLoadOp:  common.LoadOpDontCare,
			StencilStoreOp: common.StoreOpDontCare,
			InitialLayout:  initialLayout,
			FinalLayout:    finalLayout,
		},
	}

	if !clear {
		attachments[0].LoadOp = common.LoadOpLoad
	}

	if depthPresent {
		attachments = append(attachments, core.AttachmentDescription{
			Format:         i.Depth.Format,
			Samples:        NumSamples,
			LoadOp:         common.LoadOpClear,
			StoreOp:        common.StoreOpStore,
			StencilLoadOp:  common.LoadOpDontCare,
			StencilStoreOp: common.StoreOpStore,
			InitialLayout:  common.LayoutUndefined,
			FinalLayout:    common.LayoutDepthStencilAttachmentOptimal,
		})

		if !clear {
			attachments[1].LoadOp = common.LoadOpDontCare
		}
	}

	renderPassOptions := &core.RenderPassOptions{
		Attachments: attachments,
		SubPasses: []core.SubPass{
			{
				BindPoint: common.BindGraphics,
				ColorAttachments: []common.AttachmentReference{
					{
						AttachmentIndex: 0,
						Layout:          common.LayoutColorAttachmentOptimal,
					},
				},
			},
		},
		SubPassDependencies: []core.SubPassDependency{
			{
				SrcSubPassIndex: core.SubpassExternal,
				DstSubPassIndex: 0,
				SrcStageMask:    common.PipelineStageColorAttachmentOutput,
				DstStageMask:    common.PipelineStageColorAttachmentOutput,
				SrcAccessMask:   0,
				DstAccessMask:   common.AccessColorAttachmentWrite,
			},
		},
	}

	if depthPresent {
		renderPassOptions.SubPasses[0].DepthStencilAttachment = &common.AttachmentReference{
			AttachmentIndex: 1,
			Layout:          common.LayoutDepthStencilAttachmentOptimal,
		}
	}

	var err error
	i.RenderPass, _, err = i.Loader.CreateRenderPass(i.Device, renderPassOptions)
	return err
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

func (i *SampleInfo) InitShaders(vertShaderBytes []byte, fragShaderBytes []byte) error {
	vertShaderModule, _, err := i.Loader.CreateShaderModule(i.Device, &core.ShaderModuleOptions{
		SpirVByteCode: bytesToBytecode(vertShaderBytes),
	})
	if err != nil {
		return err
	}

	fragShaderModule, _, err := i.Loader.CreateShaderModule(i.Device, &core.ShaderModuleOptions{
		SpirVByteCode: bytesToBytecode(fragShaderBytes),
	})
	if err != nil {
		return err
	}

	i.ShaderStages = []*core.ShaderStage{
		{
			Stage:  common.StageVertex,
			Name:   "main",
			Shader: vertShaderModule,
		},
		{
			Stage:  common.StageFragment,
			Name:   "main",
			Shader: fragShaderModule,
		},
	}

	return nil
}

func (i *SampleInfo) InitFramebuffers(depthPresent bool) error {
	framebufferOptions := &core.FramebufferOptions{
		RenderPass:  i.RenderPass,
		Attachments: []core.ImageView{nil},
		Width:       i.Width,
		Height:      i.Height,
		Layers:      1,
	}

	if depthPresent {
		framebufferOptions.Attachments = append(framebufferOptions.Attachments, i.Depth.View)
	}

	for swapchainInd := 0; swapchainInd < i.SwapchainImageCount; swapchainInd++ {
		framebufferOptions.Attachments[0] = i.Buffers[swapchainInd].View

		var err error
		frameBuffer, _, err := i.Loader.CreateFrameBuffer(i.Device, framebufferOptions)
		if err != nil {
			return err
		}

		i.Framebuffer = append(i.Framebuffer, frameBuffer)
	}

	return nil
}

func (i *SampleInfo) InitVertexBuffers(vertexData interface{}, dataSize int, dataStride int, useTexture bool) error {
	var err error
	i.VertexBuffer.Buf, _, err = i.Loader.CreateBuffer(i.Device, &core.BufferOptions{
		BufferSize:  dataSize,
		Usage:       common.UsageVertexBuffer,
		SharingMode: common.SharingExclusive,
	})
	if err != nil {
		return err
	}

	memReqs := i.VertexBuffer.Buf.MemoryRequirements()
	memoryIndex, err := i.MemoryTypeFromProperties(memReqs.MemoryType, core.MemoryHostVisible|core.MemoryHostCoherent)
	if err != nil {
		return err
	}

	i.VertexBuffer.Mem, _, err = i.Device.AllocateMemory(&core.DeviceMemoryOptions{
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memoryIndex,
	})
	if err != nil {
		return err
	}

	i.VertexBuffer.BufferInfo.Range = memReqs.Size
	i.VertexBuffer.BufferInfo.Offset = 0

	vertexPtr, _, err := i.VertexBuffer.Mem.MapMemory(0, memReqs.Size, 0)
	if err != nil {
		return err
	}

	dataBuffer := unsafe.Slice((*byte)(vertexPtr), memReqs.Size)

	buf := &bytes.Buffer{}
	err = binary.Write(buf, common.ByteOrder, vertexData)
	if err != nil {
		i.VertexBuffer.Mem.UnmapMemory()
		return err
	}

	copy(dataBuffer, buf.Bytes())

	i.VertexBuffer.Mem.UnmapMemory()

	_, err = i.VertexBuffer.Buf.BindBufferMemory(i.VertexBuffer.Mem, 0)
	if err != nil {
		return err
	}

	i.VertexBinding = core.VertexBindingDescription{
		InputRate: core.RateVertex,
		Binding:   0,
		Stride:    dataStride,
	}
	i.VertexAttributes = []core.VertexAttributeDescription{
		{
			Binding:  0,
			Location: 0,
			Offset:   0,
			Format:   common.FormatR32G32B32A32SignedFloat,
		},
		{
			Binding:  0,
			Location: 1,
			Offset:   16,
		},
	}

	if useTexture {
		i.VertexAttributes[1].Format = common.FormatR32G32SignedFloat
	} else {
		i.VertexAttributes[1].Format = common.FormatR32G32B32A32SignedFloat
	}

	return nil
}

func (i *SampleInfo) InitDescriptorPool(useTexture bool) error {
	poolSizes := []core.PoolSize{
		{
			Type:            common.DescriptorUniformBuffer,
			DescriptorCount: 1,
		},
	}

	if useTexture {
		poolSizes = append(poolSizes, core.PoolSize{
			Type:            common.DescriptorCombinedImageSampler,
			DescriptorCount: 1,
		})
	}

	var err error
	i.DescPool, _, err = i.Loader.CreateDescriptorPool(i.Device, &core.DescriptorPoolOptions{
		MaxSets:   1,
		PoolSizes: poolSizes,
	})

	return err
}

func (i *SampleInfo) InitDescriptorSet(useTexture bool) error {
	var err error
	i.DescSet, _, err = i.DescPool.AllocateDescriptorSets(&core.DescriptorSetOptions{
		AllocationLayouts: i.DescLayout,
	})
	if err != nil {
		log.Fatalln(err)
	}

	writes := []core.WriteDescriptorSetOptions{
		{
			DstSet:          i.DescSet[0],
			DstBinding:      0,
			DstArrayElement: 0,

			DescriptorType: common.DescriptorUniformBuffer,
			BufferInfo:     []core.DescriptorBufferInfo{i.UniformData.BufferInfo},
		},
	}

	if useTexture {
		writes = append(writes, core.WriteDescriptorSetOptions{
			DstSet:          i.DescSet[0],
			DstBinding:      1,
			DstArrayElement: 0,

			DescriptorType: common.DescriptorCombinedImageSampler,
			ImageInfo:      []core.DescriptorImageInfo{i.TextureData.ImageInfo},
		})
	}

	return i.Device.UpdateDescriptorSets(writes, nil)
}

func (i *SampleInfo) InitPipelineCache() error {
	var err error
	i.PipelineCache, _, err = i.Loader.CreatePipelineCache(i.Device, &core.PipelineCacheOptions{})
	return err
}

func (i *SampleInfo) InitPipeline(depthPresent bool, vertexPresent bool) error {
	pipelineOptions := &core.GraphicsPipelineOptions{
		ShaderStages: i.ShaderStages,
		VertexInput:  &core.VertexInputOptions{},
		InputAssembly: &core.InputAssemblyOptions{
			EnablePrimitiveRestart: false,
			Topology:               common.TopologyTriangleList,
		},
		Viewport: &core.ViewportOptions{
			Viewports: []common.Viewport{
				{},
			},
			Scissors: []common.Rect2D{
				{},
			},
		},
		Rasterization: &core.RasterizationOptions{
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
		},
		Multisample: &core.MultisampleOptions{
			RasterizationSamples: NumSamples,
			SampleShading:        false,
			AlphaToCoverage:      false,
			AlphaToOne:           false,
			MinSampleShading:     0,
		},
		DepthStencil: &core.DepthStencilOptions{
			DepthTestEnable:       depthPresent,
			DepthWriteEnable:      depthPresent,
			DepthCompareOp:        common.CompareLessOrEqual,
			DepthBoundsTestEnable: false,
			StencilTestEnable:     false,
			BackStencilState: core.StencilOpState{
				FailOp:      common.StencilKeep,
				PassOp:      common.StencilKeep,
				CompareOp:   common.CompareAlways,
				CompareMask: 0,
				Reference:   0,
				DepthFailOp: common.StencilKeep,
				WriteMask:   0,
			},
			FrontStencilState: core.StencilOpState{
				FailOp:      common.StencilKeep,
				PassOp:      common.StencilKeep,
				CompareOp:   common.CompareAlways,
				CompareMask: 0,
				Reference:   0,
				DepthFailOp: common.StencilKeep,
				WriteMask:   0,
			},
			MinDepthBounds: 0,
			MaxDepthBounds: 0,
		},
		ColorBlend: &core.ColorBlendOptions{
			LogicOpEnabled: false,
			LogicOp:        common.LogicOpNoop,
			BlendConstants: [4]float32{1, 1, 1, 1},
			Attachments: []core.ColorBlendAttachment{
				{
					BlendEnabled: false,
					SrcColor:     common.BlendZero,
					DstColor:     common.BlendZero,
					ColorBlendOp: common.BlendOpAdd,
					SrcAlpha:     common.BlendZero,
					DstAlpha:     common.BlendZero,
					AlphaBlendOp: common.BlendOpAdd,
					WriteMask:    common.ComponentRed | common.ComponentGreen | common.ComponentBlue | common.ComponentAlpha,
				},
			},
		},
		DynamicState: &core.DynamicStateOptions{
			DynamicStates: []core.DynamicState{core.DynamicStateViewport, core.DynamicStateScissor},
		},
		Layout:     i.PipelineLayout,
		RenderPass: i.RenderPass,
		SubPass:    0,
	}

	if vertexPresent {
		pipelineOptions.VertexInput.VertexBindingDescriptions = []core.VertexBindingDescription{i.VertexBinding}
		pipelineOptions.VertexInput.VertexAttributeDescriptions = i.VertexAttributes
	}

	pipelines, _, err := i.Loader.CreateGraphicsPipelines(i.Device, i.PipelineCache,
		[]*core.GraphicsPipelineOptions{
			pipelineOptions,
		})

	i.Pipeline = pipelines[0]
	return err
}

func (i *SampleInfo) InitPresentableImage() error {
	var err error
	i.ImageAcquiredSemaphore, _, err = i.Loader.CreateSemaphore(i.Device, &core.SemaphoreOptions{})
	if err != nil {
		return err
	}

	// Get the index of the next available swapchain image:
	i.CurrentBuffer, _, err = i.Swapchain.AcquireNextImage(common.NoTimeout, i.ImageAcquiredSemaphore, nil)

	// TODO: Deal with the VK_SUBOPTIMAL_KHR and VK_ERROR_OUT_OF_DATE_KHR
	// return codes
	return err
}

func (i *SampleInfo) InitClearColorAndDepth() []core.ClearValue {
	return []core.ClearValue{
		core.ClearValueFloat{0.2, 0.2, 0.2, 0.2},
		core.ClearValueDepthStencil{Depth: 1.0, Stencil: 0},
	}
}

func (i *SampleInfo) InitRenderPassBeginInfo() *core.RenderPassBeginOptions {
	return &core.RenderPassBeginOptions{
		RenderPass:  i.RenderPass,
		Framebuffer: i.Framebuffer[i.CurrentBuffer],
		RenderArea: common.Rect2D{
			Offset: common.Offset2D{0, 0},
			Extent: common.Extent2D{Width: i.Width, Height: i.Height},
		},
	}
}

func (i *SampleInfo) InitViewports() {
	i.Viewport = common.Viewport{
		X:        0,
		Y:        0,
		Width:    float32(i.Width),
		Height:   float32(i.Height),
		MinDepth: 0,
		MaxDepth: 1,
	}
	i.Cmd.CmdSetViewport([]common.Viewport{i.Viewport})
}

func (i *SampleInfo) InitScissors() {
	i.Scissor = common.Rect2D{
		Offset: common.Offset2D{0, 0},
		Extent: common.Extent2D{i.Width, i.Height},
	}
	i.Cmd.CmdSetScissor([]common.Rect2D{i.Scissor})
}

func (i *SampleInfo) InitFence() (core.Fence, error) {
	fence, _, err := i.Loader.CreateFence(i.Device, &core.FenceOptions{})
	return fence, err
}

func (i *SampleInfo) InitSubmitInfo(stageFlags common.PipelineStages) *core.SubmitOptions {
	return &core.SubmitOptions{
		CommandBuffers: []core.CommandBuffer{i.Cmd},
		WaitSemaphores: []core.Semaphore{i.ImageAcquiredSemaphore},
		WaitDstStages:  []common.PipelineStages{stageFlags},
	}
}

func (i *SampleInfo) InitPresentInfo() *khr_swapchain.PresentOptions {
	return &khr_swapchain.PresentOptions{
		Swapchains:   []khr_swapchain.Swapchain{i.Swapchain},
		ImageIndices: []int{i.CurrentBuffer},
	}
}
func (i *SampleInfo) InitSampler() (core.Sampler, error) {
	sampler, _, err := i.Loader.CreateSampler(i.Device, &core.SamplerOptions{
		MagFilter:        common.FilterNearest,
		MinFilter:        common.FilterNearest,
		MipmapMode:       common.MipmapNearest,
		AddressModeU:     common.AddressModeClampToEdge,
		AddressModeV:     common.AddressModeClampToEdge,
		AddressModeW:     common.AddressModeClampToEdge,
		MipLodBias:       0,
		AnisotropyEnable: false,
		MaxAnisotropy:    1,
		CompareOp:        common.CompareNever,
		MinLod:           0,
		MaxLod:           0,
		CompareEnable:    false,
		BorderColor:      common.BorderColorFloatOpaqueWhite,
	})

	return sampler, err
}

func (i *SampleInfo) ExecuteQueueCmdBuf(cmdBufs []core.CommandBuffer, fence core.Fence) error {
	/* Queue the command buffer for execution */
	_, err := i.GraphicsQueue.SubmitToQueue(fence, []*core.SubmitOptions{
		{
			WaitSemaphores: []core.Semaphore{i.ImageAcquiredSemaphore},
			WaitDstStages:  []common.PipelineStages{common.PipelineStageColorAttachmentOutput},
			CommandBuffers: cmdBufs,
		},
	})
	return err
}

func (i *SampleInfo) ExecutePresentImage() error {
	_, _, err := i.Swapchain.PresentToQueue(i.PresentQueue, &khr_swapchain.PresentOptions{
		Swapchains:   []khr_swapchain.Swapchain{i.Swapchain},
		ImageIndices: []int{i.CurrentBuffer},
	})
	return err
}

func (i *SampleInfo) DestroyPipeline() {
	i.Pipeline.Destroy()
}

func (i *SampleInfo) DestroyPipelineCache() {
	i.PipelineCache.Destroy()
}

func (i *SampleInfo) DestroyUniformBuffer() {
	i.UniformData.Buf.Destroy()
	i.Device.FreeMemory(i.UniformData.Mem)
}

func (i *SampleInfo) DestroyVertexBuffer() {
	i.VertexBuffer.Buf.Destroy()
	i.Device.FreeMemory(i.VertexBuffer.Mem)
}

func (i *SampleInfo) DestroyFramebuffers() {
	for ind := 0; ind < i.SwapchainImageCount; ind++ {
		i.Framebuffer[ind].Destroy()
	}
}

func (i *SampleInfo) DestroyShaders() {
	i.ShaderStages[0].Shader.Destroy()
	i.ShaderStages[1].Shader.Destroy()
}

func (i *SampleInfo) DestroyRenderpass() {
	i.RenderPass.Destroy()
}

func (i *SampleInfo) DestroyDepthBuffer() {
	i.Depth.View.Destroy()
	i.Depth.Image.Destroy()
	i.Device.FreeMemory(i.Depth.Mem)
}

func (i *SampleInfo) DestroySwapchain() {
	for j := 0; j < i.SwapchainImageCount; j++ {
		i.Buffers[j].View.Destroy()
	}

	i.Swapchain.Destroy()
}

func (i *SampleInfo) DestroyCommandBuffer() {
	i.CmdPool.FreeCommandBuffers([]core.CommandBuffer{i.Cmd})
}

func (i *SampleInfo) DestroyCommandPool() {
	i.CmdPool.Destroy()
}

func (i *SampleInfo) DestroyDevice() error {
	_, err := i.Device.WaitForIdle()
	if err != nil {
		return err
	}

	i.Device.Destroy()
	return nil
}

func (i *SampleInfo) DestroyInstance() {
	i.Instance.Destroy()
}

func (i *SampleInfo) DestroyDescriptorPool() {
	i.DescPool.Destroy()
}

func (i *SampleInfo) DestroyDescriptorAndPipelineLayouts() {
	for ind := 0; ind < NumDescriptorSets; ind++ {
		i.DescLayout[ind].Destroy()
	}
	i.PipelineLayout.Destroy()
}
