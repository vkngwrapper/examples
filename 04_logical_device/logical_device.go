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

	queueFamilyIndex := 0
	for ; queueFamilyIndex < len(queueFamilies); queueFamilyIndex++ {
		if queueFamilies[queueFamilyIndex].Flags & VKng.Graphics != 0 {
			break
		}
	}

	logicalDevice, err := app.physicalDevice.CreateDevice(app.allocator, &creation.DeviceOptions{
		QueueFamilies: []*creation.QueueFamilyOptions{
			{
				QueueFamilyIndex: queueFamilyIndex,
				QueuePriorities: []float32{1.0},
			},
		},
		EnabledFeatures: &VKng.PhysicalDeviceFeatures{},
	})
	if err != nil {
		return err
	}

	queue, err := logicalDevice.GetQueue(queueFamilyIndex, 0)
	if err != nil {
		return err
	}

	app.logicalDevice = logicalDevice
	app.queue = queue
	return nil
}
