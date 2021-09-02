package main

import (
	"github.com/CannibalVox/VKng/commands"
	"github.com/CannibalVox/VKng/core"
)

func (app *HelloTriangleApplication) createCommandBuffers(caps *PhysicalDeviceCaps) error {
	pool, err := commands.CreateCommandPool(app.allocator, app.logicalDevice, &commands.CommandPoolOptions{
		GraphicsQueueFamily: caps.GraphicsQueueFamily,
	})

	if err != nil {
		return err
	}
	app.commandPool = pool

	buffers, err := commands.CreateCommandBuffers(app.allocator, app.logicalDevice, &commands.CommandBufferOptions{
		Level:       core.LevelPrimary,
		BufferCount: len(app.swapchainImages),
		CommandPool: pool,
	})
	if err != nil {
		return err
	}
	app.commandBuffers = buffers

	for bufferIdx, buffer := range buffers {
		err = buffer.Begin(app.allocator, &commands.BeginOptions{})
		if err != nil {
			return err
		}

		err = buffer.CmdBeginRenderPass(app.allocator, commands.ContentsInline,
			&commands.RenderPassBeginOptions{
				RenderPass:  app.renderPass,
				Framebuffer: app.framebuffers[bufferIdx],
				RenderArea: core.Rect2D{
					Offset: core.Offset2D{X: 0, Y: 0},
					Extent: app.swapchainExtent,
				},
				ClearValues: []commands.ClearValue{
					commands.ClearValueFloat{0, 0, 0, 1},
				},
			})
		if err != nil {
			return err
		}

		buffer.CmdBindPipeline(core.BindGraphics, app.pipeline)
		buffer.CmdDraw(3, 1, 0, 0)
		buffer.CmdEndRenderPass()

		err = buffer.End()
		if err != nil {
			return err
		}
	}

	return nil
}
