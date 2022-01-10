package main

import (
	"bytes"
	"embed"
	"encoding/binary"
	"fmt"
	"github.com/CannibalVox/VKng/core"
	"github.com/CannibalVox/VKng/core/common"
	"github.com/CannibalVox/VKng/examples/lunarg_samples/utils"
	"github.com/CannibalVox/VKng/extensions/ext_debug_utils"
	"github.com/google/uuid"
	"github.com/loov/hrtime"
	"github.com/veandco/go-sdl2/sdl"
	"io/ioutil"
	"log"
	"os"
	"time"
	"unsafe"
)

//go:embed shaders images
var fileSystem embed.FS

func logDebug(msgType ext_debug_utils.MessageType, severity ext_debug_utils.MessageSeverity, data *ext_debug_utils.CallbackData) bool {
	log.Printf("[%s %s] - %s", severity, msgType, data.Message)
	return false
}

/*
VULKAN_SAMPLE_SHORT_DESCRIPTION
Create and use a pipeline cache across runs.
*/
// This sample tries to save and reuse pipeline cache data between runs.
// On first run, no cache will be found, it will be created and saved
// to disk. On later runs, the cache should be found, loaded, and used.
// Hopefully a speedup will observed.  In the future, the pipeline could
// be complicated a bit, to show a greater cache benefit.  Also, two
// caches could be created and merged.

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

	err = info.InitInstanceExtensionNames()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitDeviceExtensionNames()
	if err != nil {
		log.Fatalln(err)
	}

	info.InstanceExtensionNames = append(info.InstanceExtensionNames, ext_debug_utils.ExtensionName)
	info.InstanceLayerNames = append(info.InstanceLayerNames, "VK_LAYER_KHRONOS_validation")
	debugOptions := &ext_debug_utils.CreationOptions{
		CaptureSeverities: ext_debug_utils.SeverityWarning | ext_debug_utils.SeverityError,
		CaptureTypes:      ext_debug_utils.TypeAll,
		Callback:          logDebug,
	}

	err = info.InitInstance("Pipeline Cache", debugOptions)
	if err != nil {
		log.Fatalln(err)
	}

	debugLoader := ext_debug_utils.CreateLoaderFromInstance(info.Instance)
	debugMessenger, _, err := debugLoader.CreateMessenger(info.Instance, nil, debugOptions)
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitEnumerateDevice()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitSwapchainExtension()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitDevice()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitCommandPool()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitCommandBuffer()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.ExecuteBeginCommandBuffer()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitDeviceQueue()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitSwapchain(common.ImageUsageColorAttachment | common.ImageUsageTransferSrc)
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitDepthBuffer()
	if err != nil {
		log.Fatalln(err)
	}

	imageFile, err := fileSystem.Open("images/blue.png")
	if err != nil {
		log.Fatalln(err)
	}
	err = info.InitTexture(imageFile, 0, 0)
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitUniformBuffer()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitDescriptorAndPipelineLayouts(true)
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitRenderPass(true, true, common.LayoutPresentSrcKHR, common.LayoutUndefined)
	if err != nil {
		log.Fatalln(err)
	}

	vertShaderBytes, err := fileSystem.ReadFile("shaders/vert.spv")
	if err != nil {
		log.Fatalln(err)
	}

	fragShaderBytes, err := fileSystem.ReadFile("shaders/frag.spv")
	if err != nil {
		log.Fatal(err)
	}

	err = info.InitShaders(vertShaderBytes, fragShaderBytes)
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitFramebuffers(true)
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitVertexBuffers(utils.VBTextureData, binary.Size(utils.VBTextureData), int(unsafe.Sizeof(utils.VertexUV{})), true)
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitDescriptorPool(true)
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitDescriptorSet(true)
	if err != nil {
		log.Fatalln(err)
	}

	/* VULKAN_KEY_START */

	// Check disk for existing cache data
	fileName := "pipeline_cache_data.bin"
	pipelineData, fileReadErr := ioutil.ReadFile(fileName)
	if os.IsNotExist(fileReadErr) {
		fmt.Println("  Pipeline cache miss!")
	} else if fileReadErr != nil {
		log.Fatalln(fileReadErr)
	}

	if pipelineData != nil {
		//
		// Check for cache validity
		//
		// TODO: Update this as the spec evolves. The fields are not defined by the header.
		//
		// The code below supports SDK 0.10 Vulkan spec, which contains the following table:
		//
		// Offset	 Size            Meaning
		// ------    ------------    ------------------------------------------------------------------
		//      0               4    a device ID equal to VkPhysicalDeviceProperties::DeviceId written
		//                           as a stream of bytes, with the least significant byte first
		//
		//      4    VK_UUID_SIZE    a pipeline cache ID equal to VkPhysicalDeviceProperties::pipelineCacheUUID
		//
		//
		// The code must be updated for latest Vulkan spec, which contains the following table:
		//
		// Offset	 Size            Meaning
		// ------    ------------    ------------------------------------------------------------------
		//      0               4    length in bytes of the entire pipeline cache header written as a
		//                           stream of bytes, with the least significant byte first
		//      4               4    a VkPipelineCacheHeaderVersion value written as a stream of bytes,
		//                           with the least significant byte first
		//      8               4    a vendor ID equal to VkPhysicalDeviceProperties::vendorID written
		//                           as a stream of bytes, with the least significant byte first
		//     12               4    a device ID equal to VkPhysicalDeviceProperties::deviceID written
		//                           as a stream of bytes, with the least significant byte first
		//     16    VK_UUID_SIZE    a pipeline cache ID equal to VkPhysicalDeviceProperties::pipelineCacheUUID
		//

		var headerLength, vendorID, deviceID uint32
		var cacheHeaderVersion common.PipelineCacheHeaderVersion
		pipelineReader := bytes.NewReader(pipelineData)

		err = binary.Read(pipelineReader, common.ByteOrder, &headerLength)
		if err != nil {
			log.Fatalln(err)
		}

		err = binary.Read(pipelineReader, common.ByteOrder, &cacheHeaderVersion)
		if err != nil {
			log.Fatalln(err)
		}

		err = binary.Read(pipelineReader, common.ByteOrder, &vendorID)
		if err != nil {
			log.Fatalln(err)
		}

		err = binary.Read(pipelineReader, common.ByteOrder, &deviceID)
		if err != nil {
			log.Fatalln(err)
		}

		var cacheUUID uuid.UUID
		err = binary.Read(pipelineReader, common.ByteOrder, &cacheUUID)
		if err != nil {
			log.Fatalln(err)
		}

		var badCache bool

		if headerLength <= 0 {
			badCache = true
			fmt.Printf("  Bad header length in %s.\n", fileName)
			fmt.Printf("    Cache contains: 0x%x", headerLength)
		}

		if cacheHeaderVersion != common.PipelineCacheHeaderVersion1 {
			badCache = true
			fmt.Printf("  Unsupported cache header version in %s.\n", fileName)
			fmt.Printf("    Cache contains: 0x%x", cacheHeaderVersion)
		}

		if vendorID != info.GpuProps.VendorID {
			badCache = true
			fmt.Printf("  Vendor ID mismatch in %s\n", fileName)
			fmt.Printf("    Cache contains: 0x%x", vendorID)
			fmt.Printf("    Driver expects: 0x%x", info.GpuProps.VendorID)
		}

		if deviceID != info.GpuProps.DeviceID {
			badCache = true
			fmt.Printf("  Device ID mismatch in %s.\n", fileName)
			fmt.Printf("    Cache contains: 0x%x", deviceID)
			fmt.Printf("    Driver Expects: 0x%x", info.GpuProps.DeviceID)
		}

		if cacheUUID != info.GpuProps.PipelineCacheUUID {
			badCache = true
			fmt.Printf("  UUID mismatch in %s.\n", fileName)
			fmt.Printf("    Cache contains: %s\n", cacheUUID.String())
			fmt.Printf("    Driver expects: %s\n", info.GpuProps.PipelineCacheUUID.String())
		}

		if badCache {
			// Don't submit initial cache data if any version info is incorrect
			pipelineData = nil

			// And clear out the old cache file for use in next run
			fmt.Printf("  Deleting cache entry %s to repopulate\n", fileName)
			// not important if this fails
			_ = os.Remove(fileName)
		}
	}

	// Feed the initial cache data into cache creation
	info.PipelineCache, _, err = info.Loader.CreatePipelineCache(info.Device, nil, &core.PipelineCacheOptions{
		InitialData: pipelineData,
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Time (roughly) taken to create the graphics pipeline
	start := hrtime.Now()
	err = info.InitPipeline(true, true)
	if err != nil {
		log.Fatalln(err)
	}
	elapsed := hrtime.Now() - start
	fmt.Printf("  vkCreateGraphicsPipeline: %s\n", elapsed)

	// Begin standard draw stuff
	err = info.InitPresentableImage()
	if err != nil {
		log.Fatalln(err)
	}

	clearValues := info.InitClearColorAndDepth()
	rpBegin := info.InitRenderPassBeginInfo()
	rpBegin.ClearValues = clearValues
	err = info.Cmd.CmdBeginRenderPass(core.ContentsInline, rpBegin)
	if err != nil {
		log.Fatalln(err)
	}

	info.Cmd.CmdBindPipeline(common.BindGraphics, info.Pipeline)
	info.Cmd.CmdBindDescriptorSets(common.BindGraphics, info.PipelineLayout, info.DescSet, nil)
	info.Cmd.CmdBindVertexBuffers([]core.Buffer{info.VertexBuffer.Buf}, []int{0})
	info.InitViewports()
	info.InitScissors()
	info.Cmd.CmdDraw(36, 1, 0, 0)
	info.Cmd.CmdEndRenderPass()
	_, err = info.Cmd.End()
	if err != nil {
		log.Fatalln(err)
	}

	drawFence, err := info.InitFence()
	if err != nil {
		log.Fatalln(err)
	}
	submitInfo := info.InitSubmitInfo(common.PipelineStageColorAttachmentOutput)

	/* Queue the command buffer for execution */
	_, err = info.GraphicsQueue.SubmitToQueue(drawFence, []*core.SubmitOptions{submitInfo})
	if err != nil {
		log.Fatalln(err)
	}

	/* Now present the image in the window */
	present := info.InitPresentInfo()

	/* Make sure command buffer is finished before presenting */
	for {
		res, err := drawFence.Wait(utils.FenceTimeout)
		if err != nil {
			log.Fatalln(err)
		}

		if res != core.VKTimeout {
			break
		}
	}

	_, _, err = info.Swapchain.PresentToQueue(info.PresentQueue, present)
	if err != nil {
		log.Fatalln(err)
	}

	time.Sleep(time.Second)

	if info.SaveImages {
		err = info.WritePNG("pipeline_cache")
		if err != nil {
			log.Fatalln(err)
		}
	}

	// End standard draw stuff

	// TODO: Create another pipeline cache, preferably different from the first
	// one and merge it here.  Then store the merged one.

	// Store away the cache that we've populated.  This could conceivably happen
	// earlier, depends on when the pipeline cache stops being populated
	// internally.

	endCacheData, _, err := info.PipelineCache.CacheData()
	if err != nil {
		log.Fatalln(err)
	}

	err = os.WriteFile(fileName, endCacheData, 0666)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("  cacheData written to %s\n", fileName)

	/* VULKAN_KEY_END */

	drawFence.Destroy()
	info.ImageAcquiredSemaphore.Destroy()
	info.DestroyPipeline()
	info.DestroyPipelineCache()
	info.DestroyTextures()
	info.DestroyDescriptorPool()
	info.DestroyVertexBuffer()
	info.DestroyFramebuffers()
	info.DestroyShaders()
	info.DestroyRenderpass()
	info.DestroyDescriptorAndPipelineLayouts()
	info.DestroyUniformBuffer()
	info.DestroyDepthBuffer()
	info.DestroySwapchain()
	info.DestroyCommandBuffer()
	info.DestroyCommandPool()

	err = info.DestroyDevice()
	if err != nil {
		log.Fatal(err)
	}

	info.Surface.Destroy()
	debugMessenger.Destroy()
	info.DestroyInstance()
	err = info.Window.Destroy()
	if err != nil {
		log.Fatalln(err)
	}
}
