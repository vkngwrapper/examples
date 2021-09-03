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

	vertexInput := &pipeline.VertexInputOptions{}

	inputAssembly := &pipeline.InputAssemblyOptions{
		Topology:               core.TopologyTriangleList,
		EnablePrimitiveRestart: false,
	}

	vertStage := &pipeline.ShaderStage{
		Stage:  core.StageVertex,
		Shader: vertShader,
		Name:   "main",
	}

	fragStage := &pipeline.ShaderStage{
		Stage:  core.StageFragment,
		Shader: fragShader,
		Name:   "main",
	}

	viewport := &pipeline.ViewportOptions{
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

	rasterization := &pipeline.RasterizationOptions{
		DepthClamp:        false,
		RasterizerDiscard: false,

		PolygonMode: pipeline.ModeFill,
		CullMode:    core.CullBack,
		FrontFace:   core.Clockwise,

		DepthBias: false,

		LineWidth: 1.0,
	}

	multisample := &pipeline.MultisampleOptions{
		SampleShading:        false,
		RasterizationSamples: core.Samples1,
		MinSampleShading:     1.0,
	}

	colorBlend := &pipeline.ColorBlendOptions{
		LogicOpEnabled: false,
		LogicOp:        core.LogicOpCopy,

		BlendConstants: [4]float32{0, 0, 0, 0},
		Attachments: []pipeline.ColorBlendAttachment{
			{
				BlendEnabled: false,
				WriteMask:    core.ComponentRed | core.ComponentGreen | core.ComponentBlue | core.ComponentAlpha,
			},
		},
	}

	app.pipelineLayout, err = pipeline.CreatePipelineLayout(app.allocator, app.logicalDevice, &pipeline.PipelineLayoutOptions{})
	if err != nil {
		return err
	}

	pipelines, err := pipeline.CreateGraphicsPipelines(app.allocator, app.logicalDevice, []*pipeline.Options{
		{
			ShaderStages: []*pipeline.ShaderStage{
				vertStage,
				fragStage,
			},
			VertexInput:       vertexInput,
			InputAssembly:     inputAssembly,
			Viewport:          viewport,
			Rasterization:     rasterization,
			Multisample:       multisample,
			ColorBlend:        colorBlend,
			Layout:            app.pipelineLayout,
			RenderPass:        app.renderPass,
			SubPass:           0,
			BasePipelineIndex: -1,
		},
	})
	if err != nil {
		return err
	}
	app.pipeline = pipelines[0]

	return nil
}
