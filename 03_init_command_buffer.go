package main

import (
	"github.com/CannibalVox/VKng"
	"github.com/CannibalVox/cgoalloc"
	"log"
)

func init_command_buffer() {
	defAlloc := &cgoalloc.DefaultAllocator{}
	fbAlloc, err := cgoalloc.CreateFixedBlockAllocator(defAlloc, 1024*1024, 256, 8)
	if err != nil {
		log.Fatalln(err)
	}
	alloc := cgoalloc.CreateThresholdAllocator(256, defAlloc, fbAlloc)

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
	var familyIdx int
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
			familyIdx = idx
			break
		}
	}

	if device == nil {
		log.Fatalln("Could not find a queue family in physical device 0 with graphics capability")
	}
	defer device.Destroy()

	pool, err := device.CommandPoolBuilder().
		GraphicsQueueFamilyIndex(uint32(familyIdx)).
		Build(alloc)
	if err != nil {
		log.Fatalln(err)
	}
	defer pool.Destroy()

	buffers, err := pool.CommandBufferBuilder().
		Level(VKng.Primary).
		Build(alloc, 1)

	if err != nil {
		log.Fatalln(err)
	}
	defer VKng.DestroyBuffers(buffers)
}
