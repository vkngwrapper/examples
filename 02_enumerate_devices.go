package main

import (
	"github.com/CannibalVox/VKng"
	"log"
)

func enumerate_devices() {
	alloc := &VKng.DefaultAllocator{}
	i, err := (&VKng.InstanceBuilder{}).
		ApplicationName("Hello Triangle").
		ApplicationVersion(1, 0, 0).
		EngineName("No Engine").
		EngineVersion(1, 0, 0).
		Build(alloc)
	if err != nil {
		log.Fatalln(err)
	}
	defer i.Destroy()

	physicalDevices, err := i.PhysicalDevices(alloc)
	if err != nil {
		log.Fatalln(err)
	}

	queueFamilyProps, err := physicalDevices[0].QueueFamilyProperties(alloc)
	if err != nil {
		log.Fatalln(err)
	}

	var device *VKng.Device
	for idx, queueFamily := range queueFamilyProps {
		if (queueFamily.Flags & VKng.Graphics) != 0 {
			device, err = physicalDevices[0].DeviceBuilder().
				AddQueueFamily(uint32(idx)).
					AddQueuePriority(0.0).
					Complete().
				Build(alloc)

			if err != nil {
				log.Fatalln(err)
			}
			break
		}
	}

	if device == nil {
		log.Fatalln("Could not find a queue family in physical device 0 with graphics capability")
	}
	defer device.Destroy()
}
