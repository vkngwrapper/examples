package main

import (
	"github.com/CannibalVox/VKng"
	"github.com/CannibalVox/VKng/core"
	"github.com/CannibalVox/VKng/ext_surface"
	"github.com/CannibalVox/VKng/ext_swapchain"
)

func (app *HelloTriangleApplication) createSwapchain(caps *PhysicalDeviceCaps) error {
	var bestFormat *ext_surface.SurfaceFormat
	var bestFormatValue = -1

	// Find best surface format
	for _, format := range caps.SurfaceFormats {
		if bestFormatValue < 2 && format.ColorSpace == ext_surface.SRGBNonlinear && format.Format == core.FormatB8G8R8A8SRGB {
			bestFormatValue = 2
			bestFormat = format
		} else if bestFormatValue < 1 && format.ColorSpace == ext_surface.SRGBNonlinear {
			bestFormatValue = 1
			bestFormat = format
		} else if bestFormatValue < 0 {
			bestFormatValue = 0
			bestFormat = format
		}
	}

	// Find best present mode
	var presentMode = ext_surface.FIFO
	for _, mode := range caps.SurfacePresentModes {
		if mode == ext_surface.Mailbox {
			presentMode = mode
		}
	}

	var swapExtent core.Extent2D

	if caps.SurfaceCaps.CurrentExtent.Width != (^uint32(0)) {
		swapExtent.Width = caps.SurfaceCaps.CurrentExtent.Width
		swapExtent.Height = caps.SurfaceCaps.CurrentExtent.Height
	} else {
		widthInt, heightInt := app.window.VulkanGetDrawableSize()
		width := uint32(widthInt)
		height := uint32(heightInt)

		if width < caps.SurfaceCaps.MinImageExtent.Width {
			width = caps.SurfaceCaps.MinImageExtent.Width
		}
		if width > caps.SurfaceCaps.MaxImageExtent.Width {
			width = caps.SurfaceCaps.MaxImageExtent.Width
		}
		if height < caps.SurfaceCaps.MinImageExtent.Height {
			height = caps.SurfaceCaps.MinImageExtent.Height
		}
		if height > caps.SurfaceCaps.MaxImageExtent.Height {
			height = caps.SurfaceCaps.MaxImageExtent.Height
		}

		swapExtent.Width = width
		swapExtent.Height = height
	}

	swapDepth := caps.SurfaceCaps.MinImageCount + 1
	if caps.SurfaceCaps.MaxImageCount > 0 && caps.SurfaceCaps.MaxImageCount < swapDepth {
		swapDepth = caps.SurfaceCaps.MaxImageCount
	}

	sharingMode := core.SharingExclusive
	var queueFamilyIndices []int

	if *caps.GraphicsQueueFamily != *caps.PresentQueueFamily {
		sharingMode = core.SharingConcurrent
		queueFamilyIndices = append(queueFamilyIndices, *caps.GraphicsQueueFamily, *caps.PresentQueueFamily)
	}

	swapchain, err := ext_swapchain.CreateSwapchain(app.allocator, app.logicalDevice, &ext_swapchain.CreationOptions{
		Surface: app.surface,

		MinImageCount:    swapDepth,
		ImageFormat:      bestFormat.Format,
		ImageColorSpace:  bestFormat.ColorSpace,
		ImageExtent:      swapExtent,
		ImageArrayLayers: 1,
		ImageUsage:       core.UsageColorAttachment,

		SharingMode:        sharingMode,
		QueueFamilyIndices: queueFamilyIndices,

		PreTransform:   caps.SurfaceCaps.CurrentTransform,
		CompositeAlpha: ext_surface.Opaque,
		PresentMode:    presentMode,
		Clipped:        true,
	})
	if err != nil {
		return err
	}

	images, err := swapchain.Images(app.allocator)
	if err != nil {
		return err
	}

	var imageViews []*VKng.ImageView
	for _, image := range images {
		view, err := app.logicalDevice.CreateImageView(app.allocator, &VKng.ImageViewOptions{
			ViewType: core.View2D,
			Image:    image,
			Format:   bestFormat.Format,
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

	app.swapchainExtent = swapExtent
	app.swapchain = swapchain
	app.swapchainImages = images
	app.swapchainImageViews = imageViews
	return nil
}
