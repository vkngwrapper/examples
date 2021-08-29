package main

import (
	"fmt"
	"github.com/CannibalVox/VKng"
	"github.com/CannibalVox/VKng/core"
	"github.com/CannibalVox/VKng/ext_surface"
	"github.com/CannibalVox/VKng/ext_swapchain"
	"github.com/CannibalVox/cgoalloc"
	"github.com/palantir/stacktrace"
)

type PhysicalDeviceCaps struct {
	Properties *core.PhysicalDeviceProperties
	Features *core.PhysicalDeviceFeatures

	GraphicsQueueFamily *int
	PresentQueueFamily *int

	Extensions map[string]*core.ExtensionProperties

	SurfaceCaps *ext_surface.Capabilities
	SurfaceFormats []*ext_surface.SurfaceFormat
	SurfacePresentModes []ext_surface.PresentMode
}

func (c *PhysicalDeviceCaps) Suitability() int {
	if !c.Features.GeometryShader { return 0 }
	if c.GraphicsQueueFamily == nil || c.PresentQueueFamily == nil { return 0 }
	if c.SurfaceCaps == nil { return 0 }
	if len(c.SurfaceFormats) == 0 { return 0 }
	if len(c.SurfacePresentModes) == 0 {return 0}

	score := int(c.Properties.Limits.MaxImageDimension2D)
	if c.Properties.Type == core.DiscreteGPU {
		score += 1000
	}

	return score
}

func CreatePhysicalDeviceCaps(allocator cgoalloc.Allocator, device *VKng.PhysicalDevice, surface *ext_surface.Surface) (*PhysicalDeviceCaps, error) {
	var err error
	caps := &PhysicalDeviceCaps{}

	caps.Properties, err = device.Properties(allocator)
	if err != nil {
		return nil, err
	}

	caps.Features, err = device.Features(allocator)
	if err != nil {
		return nil, err
	}

	caps.Extensions, err = device.AvailableExtensions(allocator)
	if err != nil {
		return nil, err
	}

	queueFamilies, err := device.QueueFamilyProperties(allocator)
	if err != nil {
		return nil, err
	}
	for queueFamilyIdx, queueFamily := range queueFamilies {
		if caps.GraphicsQueueFamily == nil && (queueFamily.Flags &core.Graphics) != 0 {
			gfxIdx := queueFamilyIdx
			caps.GraphicsQueueFamily = &gfxIdx
		}
		if caps.PresentQueueFamily == nil && surface.SupportsDevice(device, queueFamilyIdx) {
			presentIdx := queueFamilyIdx
			caps.PresentQueueFamily = &presentIdx
		}
	}

	// If swapchain extension is available, query surface info
	_, hasSwapchain := caps.Extensions[ext_swapchain.ExtensionName]
	if hasSwapchain {
		caps.SurfaceCaps, err = surface.Capabilities(allocator, device)
		if err != nil {
			return nil, err
		}

		caps.SurfacePresentModes, err = surface.PresentModes(allocator, device)
		if err != nil {
			return nil, err
		}

		caps.SurfaceFormats, err = surface.Formats(allocator, device)
		if err != nil {
			return nil, err
		}
	}

	return caps, nil
}

func (app *HelloTriangleApplication) pickPhysicalDevice() (*PhysicalDeviceCaps, error) {
	physicalDevices, err := app.instance.PhysicalDevices(app.allocator)
	if err != nil {
		return nil, err
	}

	bestScore := 0
	var bestDevice *VKng.PhysicalDevice
	var bestCaps *PhysicalDeviceCaps

	for _, device := range physicalDevices {
		caps, err := CreatePhysicalDeviceCaps(app.allocator, device, app.surface)
		if err != nil {
			fmt.Printf("could not pull physical device capabilities: %v\n", err)
			continue
		}
		score := caps.Suitability()

		if score > bestScore {
			bestScore = score
			bestDevice = device
			bestCaps = caps
		}
	}

	if bestDevice == nil {
		return nil, stacktrace.NewError("failed to find a suitable GPU!")
	}

	app.physicalDevice = bestDevice

	return bestCaps, nil
}
