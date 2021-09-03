package main

import (
	"github.com/CannibalVox/VKng"
	"github.com/CannibalVox/VKng/core"
	"github.com/CannibalVox/VKng/ext_swapchain"
)

func (app *HelloTriangleApplication) createLogicalDevice(caps *PhysicalDeviceCaps) error {
	graphicsFamily := *caps.GraphicsQueueFamily
	presentFamily := *caps.PresentQueueFamily

	var queueFamilyOptions []*VKng.QueueFamilyOptions
	queueFamilyOptions = append(queueFamilyOptions, &VKng.QueueFamilyOptions{
		QueueFamilyIndex: graphicsFamily,
		QueuePriorities:  []float32{1.0},
	})

	if graphicsFamily != presentFamily {
		queueFamilyOptions = append(queueFamilyOptions, &VKng.QueueFamilyOptions{
			QueueFamilyIndex: presentFamily,
			QueuePriorities:  []float32{1.0},
		})
	}

	logicalDevice, err := app.physicalDevice.CreateDevice(app.allocator, &VKng.DeviceOptions{
		QueueFamilies:   queueFamilyOptions,
		EnabledFeatures: &core.PhysicalDeviceFeatures{},
		ExtensionNames: []string{
			ext_swapchain.ExtensionName,
		},
	})
	if err != nil {
		return err
	}

	graphicsQueue, err := logicalDevice.GetQueue(graphicsFamily, 0)
	if err != nil {
		return err
	}

	presentationQueue, err := logicalDevice.GetQueue(presentFamily, 0)
	if err != nil {
		return err
	}

	app.logicalDevice = logicalDevice
	app.graphicsQueue = graphicsQueue
	app.presentQueue = presentationQueue
	return nil
}
