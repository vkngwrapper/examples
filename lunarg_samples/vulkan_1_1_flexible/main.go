package main

import (
	"fmt"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/vkngwrapper/core/v2"
	"github.com/vkngwrapper/core/v2/common"
	"github.com/vkngwrapper/core/v2/core1_0"
	"github.com/vkngwrapper/core/v2/core1_1"
	"github.com/vkngwrapper/examples/lunarg_samples/utils"
	"github.com/vkngwrapper/extensions/v2/khr_portability_enumeration"
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
	fmt.Printf("Loader/Runtime support detected for Vulkan %s\n", info.Loader.APIVersion())

	actualVersion := common.Vulkan1_1
	if info.Loader.APIVersion().IsAtLeast(desiredVersion) {
		extensions, _, err := info.Loader.AvailableExtensions()
		if err != nil {
			log.Fatalln(err)
		}

		var extensionList []string
		var flags core1_0.InstanceCreateFlags

		_, ok := extensions[khr_portability_enumeration.ExtensionName]
		if ok {
			extensionList = append(extensionList, khr_portability_enumeration.ExtensionName)
			flags = khr_portability_enumeration.InstanceCreateEnumeratePortability
		}

		instance, _, err := info.Loader.CreateInstance(nil, core1_0.InstanceCreateInfo{
			ApplicationName:       "vulkan_1_1_sampler",
			ApplicationVersion:    common.CreateVersion(1, 0, 0),
			EngineName:            "vulkan_1_1_sampler",
			EngineVersion:         common.CreateVersion(1, 0, 0),
			APIVersion:            desiredVersion,
			EnabledExtensionNames: extensionList,
			Flags:                 flags,
		})
		if err != nil {
			log.Fatalln(err)
		}
		defer instance.Destroy(nil)

		instance11 := core1_1.PromoteInstance(instance)
		if instance11 == nil {
			log.Fatalln("instance v1.1 not loaded")
		}

		physicalDevices, _, err := instance.EnumeratePhysicalDevices()
		if err != nil {
			log.Fatalln(err)
		}

		for _, device := range physicalDevices {
			if device.DeviceAPIVersion().IsAtLeast(desiredVersion) {
				device11 := core1_1.PromotePhysicalDevice(device)
				if device11 == nil {
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
		log.Println("Determined that this system can run desired Vulkan API version", desiredVersion)
	}
}
