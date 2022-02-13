package vulkan_1_1_flexible

import (
	"fmt"
	"github.com/CannibalVox/VKng/core/common"
	"github.com/CannibalVox/VKng/core/core1_0/loader"
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

	info.Loader, err = loader.CreateLoaderFromProcAddr(sdl.VulkanGetVkGetInstanceProcAddr())
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitGlobalLayerProperties()
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("Loader/Runtime support detected for Vulkan %s\n", info.Loader.Version())

	if info.Loader.Version().IsAtLeast(common.Vulkan1_1) {

	}
}
