package main

import (
	"embed"
	"github.com/CannibalVox/VKng"
	"github.com/CannibalVox/VKng/core"
	"github.com/CannibalVox/VKng/pipeline"
)

//go:embed shaders
var shaders embed.FS

func bytesToBytecode(b []byte) []uint32 {
	byteCode := make([]uint32, len(b)/4)
	for i := 0; i < len(byteCode); i++ {
		byteIndex := i * 4
		byteCode[i] = 0
		byteCode[i] |= uint32(b[byteIndex])
		byteCode[i] |= uint32(b[byteIndex+1]) << 8
		byteCode[i] |= uint32(b[byteIndex+2]) << 16
		byteCode[i] |= uint32(b[byteIndex+3]) << 24
	}

	return byteCode
}

func (app *HelloTriangleApplication) createGraphicsPipeline() error {
	// Load vertex shader
	vertShaderBytes, err := shaders.ReadFile("shaders/vert.spv")
	if err != nil {
		return err
	}

	createInfo := &VKng.ShaderModuleOptions{
		SpirVByteCode: bytesToBytecode(vertShaderBytes),
	}
	vertShader, err := app.logicalDevice.CreateShaderModule(app.allocator, createInfo)
	if err != nil {
		return err
	}
	defer vertShader.Destroy()

	// Load fragment shader
	fragShaderBytes, err := shaders.ReadFile("shaders/frag.spv")
	if err != nil {
		return err
	}

	createInfo = &VKng.ShaderModuleOptions{
		SpirVByteCode: bytesToBytecode(fragShaderBytes),
	}
	fragShader, err := app.logicalDevice.CreateShaderModule(app.allocator, createInfo)
	if err != nil {
		return err
	}
	defer fragShader.Destroy()

	_ = &pipeline.VertexInputOptions{}

	_ = &pipeline.InputAssemblyOptions{
		Topology:               core.TopologyTriangleList,
		EnablePrimitiveRestart: false,
	}

	_ = &pipeline.ShaderStage{
		Stage:  core.StageVertex,
		Shader: vertShader,
		Name:   "main",
	}

	_ = &pipeline.ShaderStage{
		Stage:  core.StageFragment,
		Shader: fragShader,
		Name:   "main",
	}

	_ = &pipeline.ViewportOptions{
		Viewports: []core.Viewport{
			{
				X:        0,
				Y:        0,
				Width:    float32(app.swapchainExtent.Width),
				Height:   float32(app.swapchainExtent.Height),
				MinDepth: 0,
				MaxDepth: 1,
			},
		},
		Scissors: []core.Rect2D{
			{
				Offset: core.Offset2D{X: 0, Y: 0},
				Extent: app.swapchainExtent,
			},
		},
	}

	_ = &pipeline.RasterizationOptions{
		DepthClamp:        false,
		RasterizerDiscard: false,

		PolygonMode: pipeline.ModeFill,
		CullMode:    core.CullBack,
		FrontFace:   core.Clockwise,

		DepthBias: false,

		LineWidth: 1.0,
	}

	_ = &pipeline.MultisampleOptions{
		SampleShading:        false,
		RasterizationSamples: core.Samples1,
		MinSampleShading:     1.0,
	}

	_ = &pipeline.ColorBlendOptions{
		LogicOpEnabled: false,
		LogicOp:        core.LogicOpCopy,

		BlendConstants: [4]float32{0, 0, 0, 0},
		Attachments: []pipeline.ColorBlendAttachment{
			{
				BlendEnabled: true,
				SrcColor:     core.BlendSrcAlpha,
				DstColor:     core.BlendOneMinusSrcAlpha,
				ColorBlendOp: core.BlendOpAdd,

				SrcAlpha:     core.BlendOne,
				DstAlpha:     core.BlendZero,
				AlphaBlendOp: core.BlendOpAdd,

				WriteMask: core.ComponentRed | core.ComponentGreen | core.ComponentBlue | core.ComponentAlpha,
			},
		},
	}

	app.pipelineLayout, err = pipeline.CreatePipelineLayout(app.allocator, app.logicalDevice, &pipeline.PipelineLayoutOptions{})
	if err != nil {
		return err
	}

	return nil
}
