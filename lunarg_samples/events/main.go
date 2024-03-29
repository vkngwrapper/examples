package main

import (
	"fmt"
	"github.com/vkngwrapper/core/v2"
	"github.com/vkngwrapper/core/v2/core1_0"
	"github.com/vkngwrapper/examples/lunarg_samples/utils"
	"log"
)

/*
VULKAN_SAMPLE_SHORT_DESCRIPTION
Use basic events
*/

func main() {
	loader, err := core.CreateSystemLoader()
	if err != nil {
		log.Fatalln(err)
	}

	info := &utils.SampleInfo{
		Loader: loader,
	}
	err = info.InitGlobalLayerProperties()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitDeviceExtensionNames()
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitInstance("Events", nil)
	if err != nil {
		log.Fatalln(err)
	}

	err = info.InitEnumerateDevice()
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

	/* VULKAN_KEY_START */

	// Start with a trivial command buffer and make sure fence wait doesn't time out
	info.Cmd.CmdSetViewport([]core1_0.Viewport{
		{
			X:        0,
			Y:        0,
			Width:    10,
			Height:   10,
			MinDepth: 0,
			MaxDepth: 1,
		},
	})
	err = info.ExecuteEndCommandBuffer()
	if err != nil {
		log.Fatalln(err)
	}

	fence, _, err := info.Device.CreateFence(nil, core1_0.FenceCreateInfo{})
	if err != nil {
		log.Fatalln(err)
	}

	submitInfo := core1_0.SubmitInfo{
		CommandBuffers: []core1_0.CommandBuffer{info.Cmd},
	}
	_, err = info.GraphicsQueue.Submit(fence, []core1_0.SubmitInfo{
		submitInfo,
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Make sure timeout is long enough for a simple command buffer without
	// waiting for an event
	timeouts := -1
	for {
		res, err := fence.Wait(utils.FenceTimeout)
		if err != nil {
			log.Fatalln(err)
		}

		timeouts++
		if res != core1_0.VKTimeout {
			break
		}
	}

	if timeouts != 0 {
		log.Fatalln("Unsuitable timeout value, exiting")
	}

	_, err = info.Cmd.Reset(0)
	if err != nil {
		log.Fatalln(err)
	}

	// Now create an event and wait for it on the GPU
	event, _, err := info.Device.CreateEvent(nil, core1_0.EventCreateInfo{})
	if err != nil {
		log.Fatalln(err)
	}

	err = info.ExecuteBeginCommandBuffer()
	if err != nil {
		log.Fatalln(err)
	}
	err = info.Cmd.CmdWaitEvents([]core1_0.Event{event}, core1_0.PipelineStageHost, core1_0.PipelineStageBottomOfPipe, nil, nil, nil)
	if err != nil {
		log.Fatalln(err)
	}
	err = info.ExecuteEndCommandBuffer()
	if err != nil {
		log.Fatalln(err)
	}
	_, err = fence.Reset()
	if err != nil {
		log.Fatalln(err)
	}

	// Note that stepping through this code in the debugger is a bad idea because the
	// GPU can TDR waiting for the event.  Execute the code from vkQueueSubmit through
	// vkSetEvent without breakpoints
	_, err = info.GraphicsQueue.Submit(fence, []core1_0.SubmitInfo{submitInfo})
	if err != nil {
		log.Fatalln(err)
	}

	// We should timeout waiting for the fence because the GPU should be waiting
	// on the event
	res, err := fence.Wait(utils.FenceTimeout)
	if err != nil {
		log.Fatalln(err)
	}
	if res != core1_0.VKTimeout {
		log.Fatalln("Didn't get expected timeout in WaitForFences, exiting")
	}

	// Set the event from the CPU and wait for the fence.  This should succeed
	// since we set the event
	_, err = event.Set()
	if err != nil {
		log.Fatalln(err)
	}
	for {
		res, err := fence.Wait(utils.FenceTimeout)
		if err != nil {
			log.Fatalln(err)
		}

		if res != core1_0.VKTimeout {
			break
		}
	}

	_, err = info.Cmd.Reset(0)
	if err != nil {
		log.Fatalln(err)
	}
	_, err = fence.Reset()
	if err != nil {
		log.Fatalln(err)
	}
	_, err = event.Reset()
	if err != nil {
		log.Fatalln(err)
	}

	// Now set the event from the GPU and wait on the CPU
	err = info.ExecuteBeginCommandBuffer()
	if err != nil {
		log.Fatalln(err)
	}
	info.Cmd.CmdSetEvent(event, core1_0.PipelineStageBottomOfPipe)
	err = info.ExecuteEndCommandBuffer()
	if err != nil {
		log.Fatalln(err)
	}

	// Look for the event on the CPU. It should be RESET since we haven't sent
	// the command buffer yet.
	res, _ = event.Status()
	if res != core1_0.VKEventReset {
		log.Fatalf("Unexpected status from event, expected %s, got %s\n", core1_0.VKEventReset, res)
	}

	// Send the command buffer and loop waiting for the event
	_, err = info.GraphicsQueue.Submit(fence, []core1_0.SubmitInfo{submitInfo})
	if err != nil {
		log.Fatalln(err)
	}

	polls := 0
	for res != core1_0.VKEventSet {
		res, err = event.Status()
		if err != nil {
			log.Fatalln(err)
		}
		polls++
	}
	fmt.Printf("%d polls to find the event set\n", polls)

	for {
		res, err = fence.Wait(utils.FenceTimeout)
		if err != nil {
			log.Fatalln(err)
		}

		if res != core1_0.VKTimeout {
			break
		}
	}

	event.Destroy(nil)
	fence.Destroy(nil)
	info.DestroyCommandBuffer()
	info.DestroyCommandPool()
	err = info.DestroyDevice()
	if err != nil {
		log.Fatalln(err)
	}
	info.DestroyInstance()
}
