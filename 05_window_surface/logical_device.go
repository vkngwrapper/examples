package main

import (
	"github.com/CannibalVox/VKng"
	"github.com/CannibalVox/VKng/core"
)

func (app *HelloTriangleApplication) createLogicalDevice() error {
	queueFamilies, err := app.physicalDevice.QueueFamilyProperties(app.allocator)
	if err != nil {
		return err
	}

	graphicsQueueFamily := -1
	presentationQueueFamily := -1
	for i := 0; i < len(queueFamilies); i++ {
		if presentationQueueFamily < 0 && app.surface.SupportsDevice(app.physicalDevice, i) {
			presentationQueueFamily = i
		}
		if graphicsQueueFamily < 0 && queueFamilies[i].Flags&core.Graphics != 0 {
			graphicsQueueFamily = i
		}
	}

	var queueFamilyOptions []*VKng.QueueFamilyOptions
	queueFamilyOptions = append(queueFamilyOptions, &VKng.QueueFamilyOptions{
		QueueFamilyIndex: graphicsQueueFamily,
		QueuePriorities:  []float32{1.0},
	})

	if graphicsQueueFamily != presentationQueueFamily {
		queueFamilyOptions = append(queueFamilyOptions, &VKng.QueueFamilyOptions{
			QueueFamilyIndex: presentationQueueFamily,
			QueuePriorities:  []float32{1.0},
		})
	}

	logicalDevice, err := app.physicalDevice.CreateDevice(app.allocator, &VKng.DeviceOptions{
		QueueFamilies:   queueFamilyOptions,
		EnabledFeatures: &core.PhysicalDeviceFeatures{},
	})
	if err != nil {
		return err
	}

	graphicsQueue, err := logicalDevice.GetQueue(graphicsQueueFamily, 0)
	if err != nil {
		return err
	}

	presentationQueue, err := logicalDevice.GetQueue(presentationQueueFamily, 0)
	if err != nil {
		return err
	}

	app.logicalDevice = logicalDevice
	app.graphicsQueue = graphicsQueue
	app.presentQueue = presentationQueue
	return nil
}
