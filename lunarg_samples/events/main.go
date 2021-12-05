package main

import (
	"fmt"
	"github.com/CannibalVox/VKng/core"
	"github.com/CannibalVox/VKng/core/common"
	"github.com/CannibalVox/VKng/examples/lunarg_samples/utils"
	"log"
)

func main() {
	loader, err := core.CreateStaticLinkedLoader()
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
	info.Cmd.CmdSetViewport(0, []common.Viewport{
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

	fence, _, err := info.Loader.CreateFence(info.Device, &core.FenceOptions{})
	if err != nil {
		log.Fatalln(err)
	}

	submitInfo := &core.SubmitOptions{
		CommandBuffers: []core.CommandBuffer{info.Cmd},
	}
	_, err = info.GraphicsQueue.SubmitToQueue(fence, []*core.SubmitOptions{
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
		if res != core.VKTimeout {
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
	event, _, err := info.Loader.CreateEvent(info.Device, &core.EventOptions{})
	if err != nil {
		log.Fatalln(err)
	}

	err = info.ExecuteBeginCommandBuffer()
	if err != nil {
		log.Fatalln(err)
	}
	err = info.Cmd.CmdWaitEvents([]core.Event{event}, common.PipelineStageHost, common.PipelineStageBottomOfPipe, nil, nil, nil)
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
	_, err = info.GraphicsQueue.SubmitToQueue(fence, []*core.SubmitOptions{submitInfo})
	if err != nil {
		log.Fatalln(err)
	}

	// We should timeout waiting for the fence because the GPU should be waiting
	// on the event
	res, err := fence.Wait(utils.FenceTimeout)
	if err != nil {
		log.Fatalln(err)
	}
	if res != core.VKTimeout {
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

		if res != core.VKTimeout {
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
	info.Cmd.CmdSetEvent(event, common.PipelineStageBottomOfPipe)
	err = info.ExecuteEndCommandBuffer()
	if err != nil {
		log.Fatalln(err)
	}

	// Look for the event on the CPU. It should be RESET since we haven't sent
	// the command buffer yet.
	res, _ = event.Status()
	if res != core.VKEventReset {
		log.Fatalf("Unexpected status from event, expected %s, got %s\n", core.VKEventReset, res)
	}

	// Send the command buffer and loop waiting for the event
	_, err = info.GraphicsQueue.SubmitToQueue(fence, []*core.SubmitOptions{submitInfo})
	if err != nil {
		log.Fatalln(err)
	}

	polls := 0
	for res != core.VKEventSet {
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

		if res != core.VKTimeout {
			break
		}
	}

	event.Destroy()
	fence.Destroy()
	info.DestroyCommandBuffer()
	info.DestroyCommandPool()
	err = info.DestroyDevice()
	if err != nil {
		log.Fatalln(err)
	}
	info.DestroyInstance()
}
