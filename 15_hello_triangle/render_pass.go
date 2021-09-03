package main

import (
	"github.com/CannibalVox/VKng"
	"github.com/CannibalVox/VKng/core"
	"github.com/CannibalVox/VKng/render_pass"
)

func (app *HelloTriangleApplication) createRenderPass() error {
	renderPass, err := render_pass.CreateRenderPass(app.allocator, app.logicalDevice, &render_pass.RenderPassOptions{
		Attachments: []render_pass.AttachmentDescription{
			{
				Format:         app.swapchainFormat.Format,
				Samples:        core.Samples1,
				LoadOp:         core.LoadOpClear,
				StoreOp:        core.StoreOpStore,
				StencilLoadOp:  core.LoadOpDontCare,
				StencilStoreOp: core.StoreOpDontCare,
				InitialLayout:  core.LayoutUndefined,
				FinalLayout:    core.LayoutPresentSrc,
			},
		},
		SubPasses: []render_pass.SubPass{
			{
				BindPoint: core.BindGraphics,
				ColorAttachments: []core.AttachmentReference{
					{
						AttachmentIndex: 0,
						Layout:          core.LayoutColorAttachmentOptimal,
					},
				},
			},
		},
		SubPassDependencies: []render_pass.SubPassDependency{
			{
				SrcSubPassIndex: render_pass.SubpassExternal,
				DstSubPassIndex: 0,

				SrcStageMask: core.PipelineStageColorAttachmentOutput,
				SrcAccess:    0,

				DstStageMask: core.PipelineStageColorAttachmentOutput,
				DstAccess:    core.AccessColorAttachmentWrite,
			},
		},
	})
	if err != nil {
		return err
	}

	app.renderPass = renderPass

	return nil
}

func (app *HelloTriangleApplication) createFramebuffers() error {
	for _, imageView := range app.swapchainImageViews {
		framebuffer, err := render_pass.CreateFrameBuffer(app.allocator, app.logicalDevice, &render_pass.FramebufferOptions{
			RenderPass: app.renderPass,
			Layers:     1,
			Attachments: []*VKng.ImageView{
				imageView,
			},
			Width:  app.swapchainExtent.Width,
			Height: app.swapchainExtent.Height,
		})
		if err != nil {
			return err
		}

		app.framebuffers = append(app.framebuffers, framebuffer)
	}

	return nil
}
