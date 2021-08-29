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

	_ = &pipeline.ShaderStageOptions{
		Stage:  core.StageVertex,
		Shader: vertShader,
		Name:   "main",
	}

	_ = &pipeline.ShaderStageOptions{
		Stage:  core.StageFragment,
		Shader: fragShader,
		Name:   "main",
	}

	return nil
}
