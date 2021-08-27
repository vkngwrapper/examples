package main

import (
	"github.com/CannibalVox/VKng"
	"github.com/CannibalVox/VKng/objects"
	"github.com/palantir/stacktrace"
	"log"
)

func (app *HelloTriangleApplication) rateDeviceSuitability(device *objects.PhysicalDevice) int {
	properties, err := device.Properties(app.allocator)
	if err != nil {
		log.Printf("could not get physical device properties: %v\n", err)
		return 0
	}
	features, err := device.Features(app.allocator)
	if err != nil {
		log.Printf("could not get physical device features: %v\n", err)
		return 0
	}

	if !features.GeometryShader {
		return 0
	}

	queueFamilies, err := device.QueueFamilyProperties(app.allocator)
	if err != nil {
		log.Printf("could not get physical device queue families: %v\n", err)
		return 0
	}

	foundGraphics := false
	for _, queueFamily := range queueFamilies {
		if queueFamily.Flags & VKng.Graphics != 0 {
			foundGraphics = true
			break
		}
	}

	if !foundGraphics {
		return 0
	}

	score := int(properties.Limits.MaxImageDimension2D)
	if properties.Type == objects.DiscreteGPU {
		score += 1000
	}

	return score
}

func (app *HelloTriangleApplication) pickPhysicalDevice() error {
	physicalDevices, err := app.instance.PhysicalDevices(app.allocator)
	if err != nil {
		return err
	}

	bestScore := 0
	var bestDevice *objects.PhysicalDevice

	for _, device := range physicalDevices {
		score := app.rateDeviceSuitability(device)

		if score > bestScore {
			bestScore = score
			bestDevice = device
		}
	}

	if bestDevice == nil {
		return stacktrace.NewError("failed to find a suitable GPU!")
	}

	app.physicalDevice = bestDevice

	return nil
}
