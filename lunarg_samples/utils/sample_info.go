package utils

import (
	"bytes"
	"encoding/binary"
	"github.com/CannibalVox/VKng/core"
	"github.com/CannibalVox/VKng/core/common"
	"github.com/CannibalVox/VKng/core/core1_0"
	"github.com/CannibalVox/VKng/extensions/khr_surface"
	"github.com/CannibalVox/VKng/extensions/khr_swapchain"
	"github.com/CannibalVox/VKng/extensions/vkng_surface_sdl2"
	"github.com/cockroachdb/errors"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/veandco/go-sdl2/sdl"
	"log"
	"unsafe"
)

type LayerProperties struct {
	Properties         *core1_0.LayerProperties
	InstanceExtensions []*core1_0.ExtensionProperties
	DeviceExtensions   []*core1_0.ExtensionProperties
}

type SwapchainBuffer struct {
	Image core1_0.Image
	View  core1_0.ImageView
}

type SampleInfo struct {
	Loader           core.Loader
	Window           *sdl.Window
	Surface          khr_surface.Surface
	Prepared         bool
	UseStagingBuffer bool
	SaveImages       bool

	InstanceLayerNames          []string
	InstanceExtensionNames      []string
	InstanceLayerProperties     []*LayerProperties
	InstanceExtensionProperties []*core1_0.ExtensionProperties
	Instance                    core1_0.Instance

	DeviceExtensionNames      []string
	DeviceExtensionProperties []*core1_0.ExtensionProperties
	Gpus                      []core1_0.PhysicalDevice
	Device                    core1_0.Device
	GraphicsQueue             core1_0.Queue
	PresentQueue              core1_0.Queue
	GraphicsQueueFamilyIndex  int
	PresentQueueFamilyIndex   int
	GpuProps                  *core1_0.PhysicalDeviceProperties
	QueueProps                []*core1_0.QueueFamily
	MemoryProperties          *core1_0.PhysicalDeviceMemoryProperties

	Framebuffer   []core1_0.Framebuffer
	Width, Height int
	Format        core1_0.Format

	SwapchainImageCount    int
	SwapchainExtension     khr_swapchain.Extension
	Swapchain              khr_swapchain.Swapchain
	Buffers                []SwapchainBuffer
	ImageAcquiredSemaphore core1_0.Semaphore

	CmdPool core1_0.CommandPool

	Depth struct {
		Format core1_0.Format
		Image  core1_0.Image
		Mem    core1_0.DeviceMemory
		View   core1_0.ImageView
	}

	Textures []*TextureObject

	UniformData struct {
		Buf        core1_0.Buffer
		Mem        core1_0.DeviceMemory
		BufferInfo core1_0.DescriptorBufferInfo
	}

	TextureData struct {
		ImageInfo core1_0.DescriptorImageInfo
	}

	VertexBuffer struct {
		Buf        core1_0.Buffer
		Mem        core1_0.DeviceMemory
		BufferInfo core1_0.DescriptorBufferInfo
	}

	VertexBinding    core1_0.VertexInputBindingDescription
	VertexAttributes []core1_0.VertexInputAttributeDescription

	Projection mgl32.Mat4
	View       mgl32.Mat4
	Model      mgl32.Mat4
	Clip       mgl32.Mat4
	MVP        mgl32.Mat4

	Cmd            core1_0.CommandBuffer // BUffer for initialization commands
	PipelineLayout core1_0.PipelineLayout
	DescLayout     []core1_0.DescriptorSetLayout
	PipelineCache  core1_0.PipelineCache
	RenderPass     core1_0.RenderPass
	Pipeline       core1_0.Pipeline

	ShaderStages []core1_0.PipelineShaderStageCreateInfo

	DescPool core1_0.DescriptorPool
	DescSet  []core1_0.DescriptorSet

	//PFN_vkCreateDebugReportCallbackEXT dbgCreateDebugReportCallback;
	//PFN_vkDestroyDebugReportCallbackEXT dbgDestroyDebugReportCallback;
	//PFN_vkDebugReportMessageEXT dbgBreakCallback;
	//std::vector<VkDebugReportCallbackEXT> debug_report_callbacks;

	CurrentBuffer    int
	QueueFamilyCount int

	Viewport core1_0.Viewport
	Scissor  core1_0.Rect2D
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
	i.Instance, _, err = i.Loader.CreateInstance(nil, core1_0.InstanceCreateInfo{
		ApplicationName:       appShortName,
		ApplicationVersion:    common.CreateVersion(0, 0, 1),
		EngineName:            appShortName,
		EngineVersion:         common.CreateVersion(0, 0, 1),
		APIVersion:            common.Vulkan1_0,
		EnabledExtensionNames: i.InstanceExtensionNames,
		EnabledLayerNames:     i.InstanceLayerNames,
		NextOptions: common.NextOptions{
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
	i.Gpus, _, err = i.Instance.EnumeratePhysicalDevices()
	if err != nil {
		return err
	}

	i.QueueProps = i.Gpus[0].QueueFamilyProperties()
	i.QueueFamilyCount = len(i.QueueProps)

	i.MemoryProperties = i.Gpus[0].MemoryProperties()
	i.GpuProps, err = i.Gpus[0].Properties()
	if err != nil {
		return err
	}

	for _, layerProps := range i.InstanceLayerProperties {
		err = i.InitDeviceExtensionProperties(layerProps)
		if err != nil {
			return err
		}
	}

	return nil
}

func (i *SampleInfo) InitDeviceExtensionProperties(layerProps *LayerProperties) error {
	deviceExtensions, _, err := i.Gpus[0].EnumerateDeviceExtensionPropertiesForLayer(layerProps.Properties.LayerName)
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
	surfaceLoader := vkng_surface_sdl2.CreateExtensionFromInstance(i.Instance)

	var err error
	i.Surface, _, err = surfaceLoader.CreateSurface(i.Instance, i.Window)
	if err != nil {
		return err
	}

	// Iterate over each queue to learn whether it supports presenting:
	var presentSupport []bool
	for queueIndex := range i.QueueProps {
		support, _, err := i.Surface.PhysicalDeviceSurfaceSupport(i.Gpus[0], queueIndex)
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
		if (queueFamily.QueueFlags & core1_0.QueueGraphics) != 0 {
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
	formats, _, err := i.Surface.PhysicalDeviceSurfaceFormats(i.Gpus[0])
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
	i.Device, _, err = i.Gpus[0].CreateDevice(nil, core1_0.DeviceCreateInfo{
		QueueCreateInfos: []core1_0.DeviceQueueCreateInfo{
			{
				QueueFamilyIndex: i.GraphicsQueueFamilyIndex,
				QueuePriorities:  []float32{0.0},
			},
		},
		EnabledExtensionNames: i.DeviceExtensionNames,
	})
	return err
}

func (i *SampleInfo) InitCommandPool() error {
	var err error
	i.CmdPool, _, err = i.Device.CreateCommandPool(nil, core1_0.CommandPoolCreateInfo{
		QueueFamilyIndex: &i.GraphicsQueueFamilyIndex,
		Flags:            core1_0.CommandPoolCreateResetBuffer,
	})
	return err
}

func (i *SampleInfo) InitCommandBuffer() error {
	buffers, _, err := i.Device.AllocateCommandBuffers(core1_0.CommandBufferAllocateInfo{
		CommandPool:        i.CmdPool,
		Level:              core1_0.CommandBufferLevelPrimary,
		CommandBufferCount: 1,
	})
	if err != nil {
		return err
	}
	i.Cmd = buffers[0]
	return nil
}

func (i *SampleInfo) ExecuteBeginCommandBuffer() error {
	_, err := i.Cmd.Begin(core1_0.CommandBufferBeginInfo{})
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

func (i *SampleInfo) InitSwapchain(usage core1_0.ImageUsageFlags) error {
	surfaceCaps, _, err := i.Surface.PhysicalDeviceSurfaceCapabilities(i.Gpus[0])
	if err != nil {
		return err
	}

	var swapchainExtent core1_0.Extent2D
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
	presentMode := khr_surface.PresentModeFIFO

	// Determine the number of VkImage's to use in the swap chain.
	// We need to acquire only 1 presentable image at a time.
	// Asking for minImageCount images ensures that we can acquire
	// 1 presentable image as long as we present it before attempting
	// to acquire another.
	desiredNumberOfSwapChainImages := surfaceCaps.MinImageCount

	var preTransform khr_surface.SurfaceTransformFlags
	if (surfaceCaps.SupportedTransforms & khr_surface.TransformIdentity) != 0 {
		preTransform = khr_surface.TransformIdentity
	} else {
		preTransform = surfaceCaps.CurrentTransform
	}

	// Find a supported composite alpha mode - one of these is guaranteed to be set
	compositeAlpha := khr_surface.CompositeAlphaOpaque
	compositeAlphaFlags := [4]khr_surface.CompositeAlphaFlags{
		khr_surface.CompositeAlphaOpaque,
		khr_surface.CompositeAlphaPreMultiplied,
		khr_surface.CompositeAlphaPostMultiplied,
		khr_surface.CompositeAlphaInherit,
	}

	for i := 0; i < len(compositeAlphaFlags); i++ {
		if (surfaceCaps.SupportedCompositeAlpha & compositeAlphaFlags[i]) != 0 {
			compositeAlpha = compositeAlphaFlags[i]
			break
		}
	}

	i.SwapchainExtension = khr_swapchain.CreateExtensionFromDevice(i.Device)
	swapchainOptions := khr_swapchain.SwapchainCreateInfo{
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
		ImageSharingMode: core1_0.SharingModeExclusive,
	}

	if i.GraphicsQueueFamilyIndex != i.PresentQueueFamilyIndex {
		// If the graphics and present queues are from different queue families,
		// we either have to explicitly transfer ownership of images between the
		// queues, or we have to create the swapchain with imageSharingMode
		// as VK_SHARING_MODE_CONCURRENT
		swapchainOptions.ImageSharingMode = core1_0.SharingModeConcurrent
		swapchainOptions.QueueFamilyIndices = []int{
			i.GraphicsQueueFamilyIndex,
			i.PresentQueueFamilyIndex,
		}
	}

	i.Swapchain, _, err = i.SwapchainExtension.CreateSwapchain(i.Device, nil, swapchainOptions)
	if err != nil {
		return err
	}

	images, _, err := i.Swapchain.SwapchainImages()
	if err != nil {
		return err
	}
	i.SwapchainImageCount = len(images)

	for _, image := range images {
		view, _, err := i.Device.CreateImageView(nil, core1_0.ImageViewCreateInfo{
			Image:    image,
			ViewType: core1_0.ImageViewType2D,
			Format:   i.Format,
			Components: core1_0.ComponentMapping{
				R: core1_0.ComponentSwizzleRed,
				G: core1_0.ComponentSwizzleGreen,
				B: core1_0.ComponentSwizzleBlue,
				A: core1_0.ComponentSwizzleAlpha,
			},
			SubresourceRange: core1_0.ImageSubresourceRange{
				AspectMask:     core1_0.ImageAspectColor,
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
	if i.Depth.Format == core1_0.FormatUndefined {
		i.Depth.Format = core1_0.FormatD16UnsignedNormalized
	}
	depthFormat := i.Depth.Format

	props := i.Gpus[0].FormatProperties(depthFormat)

	imageOptions := core1_0.ImageCreateOptions{
		ImageType: core1_0.ImageType2D,
		Format:    depthFormat,
		Extent: core1_0.Extent3D{
			Width:  i.Width,
			Height: i.Height,
			Depth:  1,
		},
		MipLevels:     1,
		ArrayLayers:   1,
		Samples:       NumSamples,
		InitialLayout: core1_0.ImageLayoutUndefined,
		SharingMode:   core1_0.SharingModeExclusive,
		Usage:         core1_0.ImageUsageDepthStencilAttachment,
	}
	if (props.LinearTilingFeatures & core1_0.FormatFeatureDepthStencilAttachment) != 0 {
		imageOptions.Tiling = core1_0.ImageTilingLinear
	} else if (props.OptimalTilingFeatures & core1_0.FormatFeatureDepthStencilAttachment) != 0 {
		imageOptions.Tiling = core1_0.ImageTilingOptimal
	} else {
		return errors.Newf("depth format %s unsupported", depthFormat)
	}

	var err error
	i.Depth.Image, _, err = i.Device.CreateImage(nil, imageOptions)
	if err != nil {
		return err
	}

	imageMemoryReqs := i.Depth.Image.MemoryRequirements()
	typeIndex, err := i.MemoryTypeFromProperties(imageMemoryReqs.MemoryTypeBits, core1_0.MemoryPropertyDeviceLocal)
	if err != nil {
		return err
	}

	i.Depth.Mem, _, err = i.Device.AllocateMemory(nil, core1_0.MemoryAllocateInfo{
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

	i.Depth.View, _, err = i.Device.CreateImageView(nil, core1_0.ImageViewCreateInfo{
		Image:  i.Depth.Image,
		Format: depthFormat,
		Components: core1_0.ComponentMapping{
			A: core1_0.ComponentSwizzleAlpha,
			R: core1_0.ComponentSwizzleRed,
			G: core1_0.ComponentSwizzleGreen,
			B: core1_0.ComponentSwizzleBlue,
		},
		SubresourceRange: core1_0.ImageSubresourceRange{
			AspectMask:     core1_0.ImageAspectDepth,
			BaseMipLevel:   0,
			LevelCount:     1,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
		ViewType: core1_0.ImageViewType2D,
	})
	return err
}

func (i *SampleInfo) MemoryTypeFromProperties(memoryType uint32, flags core1_0.MemoryPropertyFlags) (int, error) {
	for typeIndex, memType := range i.MemoryProperties.MemoryTypes {
		if (memoryType & 1) != 0 {
			// Type is available, does it match user properties?
			if (memType.PropertyFlags & flags) == flags {
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
	i.UniformData.Buf, _, err = i.Device.CreateBuffer(nil, core1_0.BufferCreateInfo{
		Usage:       core1_0.BufferUsageUniformBuffer,
		Size:        int(unsafe.Sizeof(i.MVP)),
		SharingMode: core1_0.SharingModeExclusive,
	})
	if err != nil {
		return err
	}

	memReqs := i.UniformData.Buf.MemoryRequirements()
	memoryTypeIndex, err := i.MemoryTypeFromProperties(memReqs.MemoryTypeBits, core1_0.MemoryPropertyHostVisible|core1_0.MemoryPropertyHostCoherent)
	if err != nil {
		return err
	}

	i.UniformData.Mem, _, err = i.Device.AllocateMemory(nil, core1_0.MemoryAllocateInfo{
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memoryTypeIndex,
	})
	if err != nil {
		return err
	}

	memPtr, _, err := i.UniformData.Mem.Map(0, memReqs.Size, 0)
	if err != nil {
		return err
	}

	dataBuffer := unsafe.Slice((*byte)(memPtr), memReqs.Size)

	buf := &bytes.Buffer{}
	err = binary.Write(buf, common.ByteOrder, i.MVP)
	if err != nil {
		i.UniformData.Mem.Unmap()
		return err
	}

	copy(dataBuffer, buf.Bytes())

	i.UniformData.Mem.Unmap()
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
	layoutBindings := []core1_0.DescriptorSetLayoutBinding{
		{
			Binding:         0,
			DescriptorType:  core1_0.DescriptorTypeUniformBuffer,
			DescriptorCount: 1,
			StageFlags:      core1_0.StageVertex,
		},
	}
	if useTexture {
		layoutBindings = append(layoutBindings, core1_0.DescriptorSetLayoutBinding{
			Binding:         1,
			DescriptorType:  core1_0.DescriptorTypeCombinedImageSampler,
			DescriptorCount: 1,
			StageFlags:      core1_0.StageFragment,
		})
	}

	layout, _, err := i.Device.CreateDescriptorSetLayout(nil, core1_0.DescriptorSetLayoutCreateInfo{
		Bindings: layoutBindings,
	})
	if err != nil {
		return err
	}

	i.DescLayout = []core1_0.DescriptorSetLayout{layout}
	i.PipelineLayout, _, err = i.Device.CreatePipelineLayout(nil, core1_0.PipelineLayoutCreateInfo{
		SetLayouts: []core1_0.DescriptorSetLayout{layout},
	})
	return err
}

func (i *SampleInfo) InitRenderPass(depthPresent, clear bool, finalLayout, initialLayout core1_0.ImageLayout) error {
	attachments := []core1_0.AttachmentDescription{
		{
			Format:         i.Format,
			Samples:        NumSamples,
			LoadOp:         core1_0.AttachmentLoadOpClear,
			StoreOp:        core1_0.AttachmentStoreOpStore,
			StencilLoadOp:  core1_0.AttachmentLoadOpDontCare,
			StencilStoreOp: core1_0.AttachmentStoreOpDontCare,
			InitialLayout:  initialLayout,
			FinalLayout:    finalLayout,
		},
	}

	if !clear {
		attachments[0].LoadOp = core1_0.AttachmentLoadOpLoad
	}

	if depthPresent {
		attachments = append(attachments, core1_0.AttachmentDescription{
			Format:         i.Depth.Format,
			Samples:        NumSamples,
			LoadOp:         core1_0.AttachmentLoadOpClear,
			StoreOp:        core1_0.AttachmentStoreOpStore,
			StencilLoadOp:  core1_0.AttachmentLoadOpDontCare,
			StencilStoreOp: core1_0.AttachmentStoreOpStore,
			InitialLayout:  core1_0.ImageLayoutUndefined,
			FinalLayout:    core1_0.ImageLayoutDepthStencilAttachmentOptimal,
		})

		if !clear {
			attachments[1].LoadOp = core1_0.AttachmentLoadOpDontCare
		}
	}

	renderPassOptions := core1_0.RenderPassCreateInfo{
		Attachments: attachments,
		Subpasses: []core1_0.SubpassDescription{
			{
				PipelineBindPoint: core1_0.PipelineBindPointGraphics,
				ColorAttachments: []core1_0.AttachmentReference{
					{
						Attachment: 0,
						Layout:     core1_0.ImageLayoutColorAttachmentOptimal,
					},
				},
			},
		},
		SubpassDependencies: []core1_0.SubpassDependency{
			{
				SrcSubpass:    core1_0.SubpassExternal,
				DstSubpass:    0,
				SrcStageMask:  core1_0.PipelineStageColorAttachmentOutput,
				DstStageMask:  core1_0.PipelineStageColorAttachmentOutput,
				SrcAccessMask: 0,
				DstAccessMask: core1_0.AccessColorAttachmentWrite,
			},
		},
	}

	if depthPresent {
		renderPassOptions.Subpasses[0].DepthStencilAttachment = &core1_0.AttachmentReference{
			Attachment: 1,
			Layout:     core1_0.ImageLayoutDepthStencilAttachmentOptimal,
		}
	}

	var err error
	i.RenderPass, _, err = i.Device.CreateRenderPass(nil, renderPassOptions)
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
	vertShaderModule, _, err := i.Device.CreateShaderModule(nil, core1_0.ShaderModuleCreateInfo{
		Code: bytesToBytecode(vertShaderBytes),
	})
	if err != nil {
		return err
	}

	fragShaderModule, _, err := i.Device.CreateShaderModule(nil, core1_0.ShaderModuleCreateInfo{
		Code: bytesToBytecode(fragShaderBytes),
	})
	if err != nil {
		return err
	}

	i.ShaderStages = []core1_0.PipelineShaderStageCreateInfo{
		{
			Stage:  core1_0.StageVertex,
			Name:   "main",
			Module: vertShaderModule,
		},
		{
			Stage:  core1_0.StageFragment,
			Name:   "main",
			Module: fragShaderModule,
		},
	}

	return nil
}

func (i *SampleInfo) InitFramebuffers(depthPresent bool) error {
	framebufferOptions := core1_0.FramebufferCreateInfo{
		RenderPass:  i.RenderPass,
		Attachments: []core1_0.ImageView{nil},
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
		frameBuffer, _, err := i.Device.CreateFramebuffer(nil, framebufferOptions)
		if err != nil {
			return err
		}

		i.Framebuffer = append(i.Framebuffer, frameBuffer)
	}

	return nil
}

func (i *SampleInfo) InitVertexBuffers(vertexData any, dataSize int, dataStride int, useTexture bool) error {
	var err error
	i.VertexBuffer.Buf, _, err = i.Device.CreateBuffer(nil, core1_0.BufferCreateInfo{
		Size:        dataSize,
		Usage:       core1_0.BufferUsageVertexBuffer,
		SharingMode: core1_0.SharingModeExclusive,
	})
	if err != nil {
		return err
	}

	memReqs := i.VertexBuffer.Buf.MemoryRequirements()
	memoryIndex, err := i.MemoryTypeFromProperties(memReqs.MemoryTypeBits, core1_0.MemoryPropertyHostVisible|core1_0.MemoryPropertyHostCoherent)
	if err != nil {
		return err
	}

	i.VertexBuffer.Mem, _, err = i.Device.AllocateMemory(nil, core1_0.MemoryAllocateInfo{
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memoryIndex,
	})
	if err != nil {
		return err
	}

	i.VertexBuffer.BufferInfo.Range = memReqs.Size
	i.VertexBuffer.BufferInfo.Offset = 0

	vertexPtr, _, err := i.VertexBuffer.Mem.Map(0, memReqs.Size, 0)
	if err != nil {
		return err
	}

	dataBuffer := unsafe.Slice((*byte)(vertexPtr), memReqs.Size)

	buf := &bytes.Buffer{}
	err = binary.Write(buf, common.ByteOrder, vertexData)
	if err != nil {
		i.VertexBuffer.Mem.Unmap()
		return err
	}

	copy(dataBuffer, buf.Bytes())

	i.VertexBuffer.Mem.Unmap()

	_, err = i.VertexBuffer.Buf.BindBufferMemory(i.VertexBuffer.Mem, 0)
	if err != nil {
		return err
	}

	i.VertexBinding = core1_0.VertexInputBindingDescription{
		InputRate: core1_0.RateVertex,
		Binding:   0,
		Stride:    dataStride,
	}
	i.VertexAttributes = []core1_0.VertexInputAttributeDescription{
		{
			Binding:  0,
			Location: 0,
			Offset:   0,
			Format:   core1_0.FormatR32G32B32A32SignedFloat,
		},
		{
			Binding:  0,
			Location: 1,
			Offset:   16,
		},
	}

	if useTexture {
		i.VertexAttributes[1].Format = core1_0.FormatR32G32SignedFloat
	} else {
		i.VertexAttributes[1].Format = core1_0.FormatR32G32B32A32SignedFloat
	}

	return nil
}

func (i *SampleInfo) InitDescriptorPool(useTexture bool) error {
	poolSizes := []core1_0.DescriptorPoolSize{
		{
			Type:            core1_0.DescriptorTypeUniformBuffer,
			DescriptorCount: 1,
		},
	}

	if useTexture {
		poolSizes = append(poolSizes, core1_0.DescriptorPoolSize{
			Type:            core1_0.DescriptorTypeCombinedImageSampler,
			DescriptorCount: 1,
		})
	}

	var err error
	i.DescPool, _, err = i.Device.CreateDescriptorPool(nil, core1_0.DescriptorPoolCreateInfo{
		MaxSets:   1,
		PoolSizes: poolSizes,
	})

	return err
}

func (i *SampleInfo) InitDescriptorSet(useTexture bool) error {
	descSet, _, err := i.Device.AllocateDescriptorSets(core1_0.DescriptorSetAllocateInfo{
		DescriptorPool: i.DescPool,
		SetLayouts:     i.DescLayout,
	})
	if err != nil {
		log.Fatalln(err)
	}

	i.DescSet = descSet

	writes := []core1_0.WriteDescriptorSet{
		{
			DstSet:          i.DescSet[0],
			DstBinding:      0,
			DstArrayElement: 0,

			DescriptorType: core1_0.DescriptorTypeUniformBuffer,
			BufferInfo:     []core1_0.DescriptorBufferInfo{i.UniformData.BufferInfo},
		},
	}

	if useTexture {
		writes = append(writes, core1_0.WriteDescriptorSet{
			DstSet:          i.DescSet[0],
			DstBinding:      1,
			DstArrayElement: 0,

			DescriptorType: core1_0.DescriptorTypeCombinedImageSampler,
			ImageInfo:      []core1_0.DescriptorImageInfo{i.TextureData.ImageInfo},
		})
	}

	return i.Device.UpdateDescriptorSets(writes, nil)
}

func (i *SampleInfo) InitPipelineCache() error {
	var err error
	i.PipelineCache, _, err = i.Device.CreatePipelineCache(nil, core1_0.PipelineCacheCreateInfo{})
	return err
}

func (i *SampleInfo) InitPipeline(depthPresent bool, vertexPresent bool) error {
	pipelineOptions := core1_0.GraphicsPipelineCreateInfo{
		Stages:           i.ShaderStages,
		VertexInputState: &core1_0.PipelineVertexInputStateCreateInfo{},
		InputAssemblyState: &core1_0.PipelineInputAssemblyStateCreateInfo{
			PrimitiveRestartEnable: false,
			Topology:               core1_0.PrimitiveTopologyTriangleList,
		},
		ViewportState: &core1_0.PipelineViewportStateCreateInfo{
			Viewports: []core1_0.Viewport{
				{},
			},
			Scissors: []core1_0.Rect2D{
				{},
			},
		},
		RasterizationState: &core1_0.PipelineRasterizationStateCreateInfo{
			PolygonMode:             core1_0.PolygonModeFill,
			CullMode:                core1_0.CullModeBack,
			FrontFace:               core1_0.FrontFaceClockwise,
			DepthClampEnable:        false,
			RasterizerDiscardEnable: false,
			DepthBiasEnable:         false,
			DepthBiasConstantFactor: 0,
			DepthBiasClamp:          0,
			DepthBiasSlopeFactor:    0,
			LineWidth:               1,
		},
		MultisampleState: &core1_0.PipelineMultisampleStateCreateInfo{
			RasterizationSamples:  NumSamples,
			SampleShadingEnable:   false,
			AlphaToCoverageEnable: false,
			AlphaToOneEnable:      false,
			MinSampleShading:      0,
		},
		DepthStencilState: &core1_0.PipelineDepthStencilStateCreateInfo{
			DepthTestEnable:       depthPresent,
			DepthWriteEnable:      depthPresent,
			DepthCompareOp:        core1_0.CompareOpLessOrEqual,
			DepthBoundsTestEnable: false,
			StencilTestEnable:     false,
			Back: core1_0.StencilOpState{
				FailOp:      core1_0.StencilKeep,
				PassOp:      core1_0.StencilKeep,
				CompareOp:   core1_0.CompareOpAlways,
				CompareMask: 0,
				Reference:   0,
				DepthFailOp: core1_0.StencilKeep,
				WriteMask:   0,
			},
			Front: core1_0.StencilOpState{
				FailOp:      core1_0.StencilKeep,
				PassOp:      core1_0.StencilKeep,
				CompareOp:   core1_0.CompareOpAlways,
				CompareMask: 0,
				Reference:   0,
				DepthFailOp: core1_0.StencilKeep,
				WriteMask:   0,
			},
			MinDepthBounds: 0,
			MaxDepthBounds: 0,
		},
		ColorBlendState: &core1_0.PipelineColorBlendStateCreateInfo{
			LogicOpEnabled: false,
			LogicOp:        core1_0.LogicOpNoop,
			BlendConstants: [4]float32{1, 1, 1, 1},
			Attachments: []core1_0.PipelineColorBlendAttachmentState{
				{
					BlendEnabled:        false,
					SrcColorBlendFactor: core1_0.BlendFactorZero,
					DstColorBlendFactor: core1_0.BlendFactorZero,
					ColorBlendOp:        core1_0.BlendOpAdd,
					SrcAlphaBlendFactor: core1_0.BlendFactorZero,
					DstAlphaBlendFactor: core1_0.BlendFactorZero,
					AlphaBlendOp:        core1_0.BlendOpAdd,
					ColorWriteMask:      core1_0.ColorComponentRed | core1_0.ColorComponentGreen | core1_0.ColorComponentBlue | core1_0.ColorComponentAlpha,
				},
			},
		},
		DynamicState: &core1_0.PipelineDynamicStateCreateInfo{
			DynamicStates: []core1_0.DynamicState{core1_0.DynamicStateViewport, core1_0.DynamicStateScissor},
		},
		Layout:     i.PipelineLayout,
		RenderPass: i.RenderPass,
		Subpass:    0,
	}

	if vertexPresent {
		pipelineOptions.VertexInputState.VertexBindingDescriptions = []core1_0.VertexInputBindingDescription{i.VertexBinding}
		pipelineOptions.VertexInputState.VertexAttributeDescriptions = i.VertexAttributes
	}

	pipelines, _, err := i.Device.CreateGraphicsPipelines(i.PipelineCache, nil,
		[]core1_0.GraphicsPipelineCreateInfo{
			pipelineOptions,
		})

	i.Pipeline = pipelines[0]
	return err
}

func (i *SampleInfo) InitPresentableImage() error {
	var err error
	i.ImageAcquiredSemaphore, _, err = i.Device.CreateSemaphore(nil, core1_0.SemaphoreCreateInfo{})
	if err != nil {
		return err
	}

	// Get the index of the next available swapchain image:
	i.CurrentBuffer, _, err = i.Swapchain.AcquireNextImage(common.NoTimeout, i.ImageAcquiredSemaphore, nil)

	// TODO: Deal with the VK_SUBOPTIMAL_KHR and VK_ERROR_OUT_OF_DATE_KHR
	// return codes
	return err
}

func (i *SampleInfo) InitClearColorAndDepth() []core1_0.ClearValue {
	return []core1_0.ClearValue{
		core1_0.ClearValueFloat{0.2, 0.2, 0.2, 0.2},
		core1_0.ClearValueDepthStencil{Depth: 1.0, Stencil: 0},
	}
}

func (i *SampleInfo) InitRenderPassBeginInfo() core1_0.RenderPassBeginInfo {
	return core1_0.RenderPassBeginInfo{
		RenderPass:  i.RenderPass,
		Framebuffer: i.Framebuffer[i.CurrentBuffer],
		RenderArea: core1_0.Rect2D{
			Offset: core1_0.Offset2D{0, 0},
			Extent: core1_0.Extent2D{Width: i.Width, Height: i.Height},
		},
	}
}

func (i *SampleInfo) InitViewports() {
	i.Viewport = core1_0.Viewport{
		X:        0,
		Y:        0,
		Width:    float32(i.Width),
		Height:   float32(i.Height),
		MinDepth: 0,
		MaxDepth: 1,
	}
	i.Cmd.CmdSetViewport([]core1_0.Viewport{i.Viewport})
}

func (i *SampleInfo) InitScissors() {
	i.Scissor = core1_0.Rect2D{
		Offset: core1_0.Offset2D{0, 0},
		Extent: core1_0.Extent2D{i.Width, i.Height},
	}
	i.Cmd.CmdSetScissor([]core1_0.Rect2D{i.Scissor})
}

func (i *SampleInfo) InitFence() (core1_0.Fence, error) {
	fence, _, err := i.Device.CreateFence(nil, core1_0.FenceCreateInfo{})
	return fence, err
}

func (i *SampleInfo) InitSubmitInfo(stageFlags core1_0.PipelineStageFlags) *core1_0.SubmitInfo {
	return &core1_0.SubmitInfo{
		CommandBuffers:   []core1_0.CommandBuffer{i.Cmd},
		WaitSemaphores:   []core1_0.Semaphore{i.ImageAcquiredSemaphore},
		WaitDstStageMask: []core1_0.PipelineStageFlags{stageFlags},
	}
}

func (i *SampleInfo) InitPresentInfo() khr_swapchain.PresentInfo {
	return khr_swapchain.PresentInfo{
		Swapchains:   []khr_swapchain.Swapchain{i.Swapchain},
		ImageIndices: []int{i.CurrentBuffer},
	}
}
func (i *SampleInfo) InitSampler() (core1_0.Sampler, error) {
	sampler, _, err := i.Device.CreateSampler(nil, core1_0.SamplerCreateInfo{
		MagFilter:        core1_0.FilterNearest,
		MinFilter:        core1_0.FilterNearest,
		MipmapMode:       core1_0.SamplerMipmapModeNearest,
		AddressModeU:     core1_0.SamplerAddressModeClampToEdge,
		AddressModeV:     core1_0.SamplerAddressModeClampToEdge,
		AddressModeW:     core1_0.SamplerAddressModeClampToEdge,
		MipLodBias:       0,
		AnisotropyEnable: false,
		MaxAnisotropy:    1,
		CompareOp:        core1_0.CompareOpNever,
		MinLod:           0,
		MaxLod:           0,
		CompareEnable:    false,
		BorderColor:      core1_0.BorderColorFloatOpaqueWhite,
	})

	return sampler, err
}

func (i *SampleInfo) ExecuteQueueCmdBuf(cmdBufs []core1_0.CommandBuffer, fence core1_0.Fence) error {
	/* Queue the command buffer for execution */
	_, err := i.GraphicsQueue.Submit(fence, []core1_0.SubmitInfo{
		{
			WaitSemaphores:   []core1_0.Semaphore{i.ImageAcquiredSemaphore},
			WaitDstStageMask: []core1_0.PipelineStageFlags{core1_0.PipelineStageColorAttachmentOutput},
			CommandBuffers:   cmdBufs,
		},
	})
	return err
}

func (i *SampleInfo) ExecutePresentImage() error {
	_, err := i.SwapchainExtension.QueuePresent(i.PresentQueue, khr_swapchain.PresentInfo{
		Swapchains:   []khr_swapchain.Swapchain{i.Swapchain},
		ImageIndices: []int{i.CurrentBuffer},
	})
	return err
}

func (i *SampleInfo) DestroyPipeline() {
	i.Pipeline.Destroy(nil)
}

func (i *SampleInfo) DestroyPipelineCache() {
	i.PipelineCache.Destroy(nil)
}

func (i *SampleInfo) DestroyUniformBuffer() {
	i.UniformData.Buf.Destroy(nil)
	i.UniformData.Mem.Free(nil)
}

func (i *SampleInfo) DestroyVertexBuffer() {
	i.VertexBuffer.Buf.Destroy(nil)
	i.VertexBuffer.Mem.Free(nil)
}

func (i *SampleInfo) DestroyFramebuffers() {
	for ind := 0; ind < i.SwapchainImageCount; ind++ {
		i.Framebuffer[ind].Destroy(nil)
	}
}

func (i *SampleInfo) DestroyShaders() {
	i.ShaderStages[0].Module.Destroy(nil)
	i.ShaderStages[1].Module.Destroy(nil)
}

func (i *SampleInfo) DestroyRenderpass() {
	i.RenderPass.Destroy(nil)
}

func (i *SampleInfo) DestroyDepthBuffer() {
	i.Depth.View.Destroy(nil)
	i.Depth.Image.Destroy(nil)
	i.Depth.Mem.Free(nil)
}

func (i *SampleInfo) DestroySwapchain() {
	for j := 0; j < i.SwapchainImageCount; j++ {
		i.Buffers[j].View.Destroy(nil)
	}

	i.Swapchain.Destroy(nil)
}

func (i *SampleInfo) DestroyCommandBuffer() {
	i.Cmd.Free()
}

func (i *SampleInfo) DestroyCommandPool() {
	i.CmdPool.Destroy(nil)
}

func (i *SampleInfo) DestroyDevice() error {
	_, err := i.Device.WaitIdle()
	if err != nil {
		return err
	}

	i.Device.Destroy(nil)
	return nil
}

func (i *SampleInfo) DestroyInstance() {
	i.Instance.Destroy(nil)
}

func (i *SampleInfo) DestroyDescriptorPool() {
	i.DescPool.Destroy(nil)
}

func (i *SampleInfo) DestroyDescriptorAndPipelineLayouts() {
	for ind := 0; ind < NumDescriptorSets; ind++ {
		i.DescLayout[ind].Destroy(nil)
	}
	i.PipelineLayout.Destroy(nil)
}
