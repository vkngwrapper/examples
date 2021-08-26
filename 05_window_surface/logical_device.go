package main

import (
	"github.com/CannibalVox/VKng"
	"github.com/CannibalVox/VKng/creation"
)

func (app *HelloTriangleApplication) createLogicalDevice() error {
	queueFamilies, err := app.physicalDevice.QueueFamilyProperties(app.allocator)
	if err != nil {
		return err
	}

	graphicsQueueFamily := -1
	presentationQueueFamily := -1
	for i := 0; i < len(queueFamilies); i++ {
		if presentationQueueFamily < 0 && app.surface.CanBePresentedBy(app.physicalDevice, i) {
			presentationQueueFamily = i
		}
		if graphicsQueueFamily < 0 && queueFamilies[i].Flags & VKng.Graphics != 0 {
			graphicsQueueFamily = i
		}
	}

	var queueFamilyOptions []*creation.QueueFamilyOptions
	queueFamilyOptions = append(queueFamilyOptions, &creation.QueueFamilyOptions{
		QueueFamilyIndex: graphicsQueueFamily,
		QueuePriorities: []float32{1.0},
	})

	if graphicsQueueFamily != presentationQueueFamily {
		queueFamilyOptions = append(queueFamilyOptions, &creation.QueueFamilyOptions{
			QueueFamilyIndex: presentationQueueFamily,
			QueuePriorities: []float32{1.0},
		})
	}

	logicalDevice, err := app.physicalDevice.CreateDevice(app.allocator, &creation.DeviceOptions{
		QueueFamilies: queueFamilyOptions,
		EnabledFeatures: &VKng.PhysicalDeviceFeatures{},
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
