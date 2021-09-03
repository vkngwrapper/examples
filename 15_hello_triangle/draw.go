package main

import (
	"github.com/CannibalVox/VKng"
	"github.com/CannibalVox/VKng/commands"
	"github.com/CannibalVox/VKng/core"
	"github.com/CannibalVox/VKng/ext_swapchain"
)

func (app *HelloTriangleApplication) createSemaphores() error {
	var err error
	app.imageAvailableSemaphore, err = app.logicalDevice.CreateSemaphore(app.allocator, &VKng.SemaphoreOptions{})
	if err != nil {
		return err
	}

	app.renderFinishedSemaphore, err = app.logicalDevice.CreateSemaphore(app.allocator, &VKng.SemaphoreOptions{})
	return err
}

func (app *HelloTriangleApplication) drawFrame() error {
	imageIndex, err := app.swapchain.AcquireNextImage(ext_swapchain.NoTimeout, app.imageAvailableSemaphore, nil)
	if err != nil {
		return err
	}

	err = commands.SubmitToQueue(app.allocator, app.graphicsQueue, nil, []*commands.SubmitOptions{
		{
			WaitSemaphores:   []*VKng.Semaphore{app.imageAvailableSemaphore},
			WaitDstStages:    []core.PipelineStages{core.PipelineStageColorAttachmentOutput},
			CommandBuffers:   []*commands.CommandBuffer{app.commandBuffers[imageIndex]},
			SignalSemaphores: []*VKng.Semaphore{app.renderFinishedSemaphore},
		},
	})
	if err != nil {
		return err
	}

	_, err = ext_swapchain.PresentToQueue(app.allocator, app.presentQueue, &ext_swapchain.PresentOptions{
		WaitSemaphores: []*VKng.Semaphore{app.renderFinishedSemaphore},
		Swapchains:     []*ext_swapchain.Swapchain{app.swapchain},
		ImageIndices:   []int{imageIndex},
	})
	if err != nil {
		return err
	}

	err = app.presentQueue.WaitForIdle()
	if err != nil {
		return err
	}

	return nil
}
