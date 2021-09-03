package main

import (
	"github.com/CannibalVox/VKng"
	"github.com/CannibalVox/VKng/commands"
	"github.com/CannibalVox/VKng/core"
	"github.com/CannibalVox/VKng/ext_swapchain"
)

const MaxFramesInFlight = 2

func (app *HelloTriangleApplication) createSemaphores() error {
	for i := 0; i < MaxFramesInFlight; i++ {
		semaphore, err := app.logicalDevice.CreateSemaphore(app.allocator, &VKng.SemaphoreOptions{})
		if err != nil {
			return err
		}

		app.imageAvailableSemaphore = append(app.imageAvailableSemaphore, semaphore)

		semaphore, err = app.logicalDevice.CreateSemaphore(app.allocator, &VKng.SemaphoreOptions{})
		if err != nil {
			return err
		}

		app.renderFinishedSemaphore = append(app.renderFinishedSemaphore, semaphore)

		fence, err := app.logicalDevice.CreateFence(app.allocator, &VKng.FenceOptions{
			Flags: VKng.FenceSignaled,
		})
		if err != nil {
			return err
		}

		app.inFlightFence = append(app.inFlightFence, fence)
	}

	for i := 0; i < len(app.swapchainImages); i++ {
		app.imagesInFlight = append(app.imagesInFlight, nil)
	}

	return nil
}

func (app *HelloTriangleApplication) drawFrame() error {
	fences := []*VKng.Fence{app.inFlightFence[app.currentFrame]}

	err := app.logicalDevice.WaitForFences(app.allocator, true, core.NoTimeout, fences)
	if err != nil {
		return err
	}

	imageIndex, err := app.swapchain.AcquireNextImage(core.NoTimeout, app.imageAvailableSemaphore[app.currentFrame], nil)
	if err != nil {
		return err
	}

	if app.imagesInFlight[imageIndex] != nil {
		err := app.logicalDevice.WaitForFences(app.allocator, true, core.NoTimeout, []*VKng.Fence{app.imagesInFlight[imageIndex]})
		if err != nil {
			return err
		}
	}
	app.imagesInFlight[imageIndex] = app.inFlightFence[app.currentFrame]

	err = app.logicalDevice.ResetFences(app.allocator, fences)
	if err != nil {
		return err
	}

	err = commands.SubmitToQueue(app.allocator, app.graphicsQueue, app.inFlightFence[app.currentFrame], []*commands.SubmitOptions{
		{
			WaitSemaphores:   []*VKng.Semaphore{app.imageAvailableSemaphore[app.currentFrame]},
			WaitDstStages:    []core.PipelineStages{core.PipelineStageColorAttachmentOutput},
			CommandBuffers:   []*commands.CommandBuffer{app.commandBuffers[imageIndex]},
			SignalSemaphores: []*VKng.Semaphore{app.renderFinishedSemaphore[app.currentFrame]},
		},
	})
	if err != nil {
		return err
	}

	_, err = ext_swapchain.PresentToQueue(app.allocator, app.presentQueue, &ext_swapchain.PresentOptions{
		WaitSemaphores: []*VKng.Semaphore{app.renderFinishedSemaphore[app.currentFrame]},
		Swapchains:     []*ext_swapchain.Swapchain{app.swapchain},
		ImageIndices:   []int{imageIndex},
	})
	if err != nil {
		return err
	}

	app.currentFrame = (app.currentFrame + 1) % MaxFramesInFlight

	return nil
}
