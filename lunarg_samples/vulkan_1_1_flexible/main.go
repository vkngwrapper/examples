package main

import (
	"fmt"
	"github.com/CannibalVox/VKng/core"
	"github.com/CannibalVox/VKng/core/common"
	"github.com/CannibalVox/VKng/core/core1_0"
	"github.com/CannibalVox/VKng/examples/lunarg_samples/utils"
	"github.com/veandco/go-sdl2/sdl"
	"log"
)

/*
VULKAN_SAMPLE_SHORT_DESCRIPTION
Determine if the current system can use Vulkan 1.1 API features
*/

func main() {
	info := &utils.SampleInfo{}
	err := info.ProcessCommandLineArgs()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitWindowSize(500, 500)
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitWindow()
	if err != nil {
		log.Fatalln(err)
	}

	info.Loader, err = core.CreateLoaderFromProcAddr(sdl.VulkanGetVkGetInstanceProcAddr())
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitGlobalLayerProperties()
	if err != nil {
		log.Fatalln(err)
	}

	desiredVersion := common.Vulkan1_1
	fmt.Printf("Loader/Runtime support detected for Vulkan %s\n", info.Loader.Version())

	actualVersion := common.Vulkan1_1
	if info.Loader.Version().IsAtLeast(desiredVersion) {
		if info.Loader.Core1_1() == nil {
			log.Fatalln("loader v1.1 not loaded")
		}

		instance, _, err := info.Loader.CreateInstance(nil, core1_0.InstanceCreateOptions{
			ApplicationName:    "vulkan_1_1_sampler",
			ApplicationVersion: common.CreateVersion(1, 0, 0),
			EngineName:         "vulkan_1_1_sampler",
			EngineVersion:      common.CreateVersion(1, 0, 0),
			VulkanVersion:      desiredVersion,
		})
		if err != nil {
			log.Fatalln(err)
		}
		defer instance.Destroy(nil)

		if instance.Core1_1() == nil {
			log.Fatalln("instance v1.1 not loaded")
		}

		physicalDevices, _, err := info.Loader.PhysicalDevices(instance)
		if err != nil {
			log.Fatalln(err)
		}

		for _, device := range physicalDevices {
			if device.APIVersion().IsAtLeast(desiredVersion) {
				if device.Core1_1() == nil {
					log.Fatalln("physical device v1.1 not loaded")
				}

				actualVersion = desiredVersion
				break
			}
		}
	}

	if actualVersion < desiredVersion {
		log.Printf("Determined that this system can only use Vulkan API version %s instead of desired version %s\n", actualVersion, desiredVersion)
	} else {
		log.Println("Determined at this system can run desired Vulkan API version", desiredVersion)
	}
}
