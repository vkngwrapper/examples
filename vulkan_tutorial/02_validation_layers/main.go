package main

import (
	"github.com/CannibalVox/VKng/core"
	"github.com/CannibalVox/VKng/core/resource"
	ext_debugutils2 "github.com/CannibalVox/VKng/extensions/debugutils"
	"github.com/CannibalVox/cgoalloc"
	"github.com/palantir/stacktrace"
	"github.com/veandco/go-sdl2/sdl"
	"log"
)

var validationLayers = []string{"VK_LAYER_KHRONOS_validation"}

const enableValidationLayers = true

type HelloTriangleApplication struct {
	allocator cgoalloc.Allocator
	window    *sdl.Window

	instance       *resource.Instance
	debugMessenger *ext_debugutils2.Messenger
}

func (app *HelloTriangleApplication) Run() error {
	err := app.initWindow()
	if err != nil {
		return err
	}

	err = app.initVulkan()
	if err != nil {
		return err
	}
	defer app.cleanup()

	return app.mainLoop()
}

func (app *HelloTriangleApplication) initWindow() error {
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		return err
	}

	window, err := sdl.CreateWindow("Vulkan", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, 800, 600, sdl.WINDOW_SHOWN|sdl.WINDOW_VULKAN)
	if err != nil {
		return err
	}
	app.window = window

	return nil
}

func (app *HelloTriangleApplication) initVulkan() error {
	err := app.createInstance()
	if err != nil {
		return err
	}

	return app.setupDebugMessenger()
}

func (app *HelloTriangleApplication) mainLoop() error {
appLoop:
	for true {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch event.(type) {
			case *sdl.QuitEvent:
				break appLoop
			}
		}
	}

	return nil
}

func (app *HelloTriangleApplication) cleanup() {
	if app.debugMessenger != nil {
		app.debugMessenger.Destroy()
	}

	if app.instance != nil {
		app.instance.Destroy()
	}

	if app.window != nil {
		app.window.Destroy()
	}
	sdl.Quit()

	app.allocator.Destroy()
}

func (app *HelloTriangleApplication) createInstance() error {
	instanceOptions := &resource.InstanceOptions{
		ApplicationName:    "Hello Triangle",
		ApplicationVersion: core.CreateVersion(1, 0, 0),
		EngineName:         "No Engine",
		EngineVersion:      core.CreateVersion(1, 0, 0),
		VulkanVersion:      core.Vulkan1_2,
	}

	// Add extensions
	sdlExtensions := app.window.VulkanGetInstanceExtensions()
	extensions, _, err := resource.AvailableExtensions(app.allocator)
	if err != nil {
		return err
	}

	for _, ext := range sdlExtensions {
		_, hasExt := extensions[ext]
		if !hasExt {
			return stacktrace.NewError("createinstance: cannot initialize sdl: missing extension %s", ext)
		}
		instanceOptions.ExtensionNames = append(instanceOptions.ExtensionNames, ext)
	}

	if enableValidationLayers {
		instanceOptions.ExtensionNames = append(instanceOptions.ExtensionNames, ext_debugutils2.ExtensionName)
	}

	// Add layers
	layers, _, err := resource.AvailableLayers(app.allocator)
	if err != nil {
		return err
	}

	if enableValidationLayers {
		for _, layer := range validationLayers {
			_, hasValidation := layers[layer]
			if !hasValidation {
				return stacktrace.NewError("createInstance: cannot add validation- layer %s not available- install LunarG Vulkan SDK", layer)
			}
			instanceOptions.LayerNames = append(instanceOptions.LayerNames, layer)
		}

		// Add debug messenger
		instanceOptions.Next = app.debugMessengerOptions()
	}

	app.instance, _, err = resource.CreateInstance(app.allocator, instanceOptions)
	if err != nil {
		return err
	}

	return nil
}

func (app *HelloTriangleApplication) debugMessengerOptions() *ext_debugutils2.Options {
	return &ext_debugutils2.Options{
		CaptureSeverities: ext_debugutils2.SeverityError | ext_debugutils2.SeverityWarning,
		CaptureTypes:      ext_debugutils2.TypeAll,
		Callback:          app.logDebug,
	}
}

func (app *HelloTriangleApplication) setupDebugMessenger() error {
	if !enableValidationLayers {
		return nil
	}

	var err error
	app.debugMessenger, _, err = ext_debugutils2.CreateMessenger(app.allocator, app.instance, app.debugMessengerOptions())
	if err != nil {
		return err
	}

	return nil
}

func (app *HelloTriangleApplication) logDebug(msgType ext_debugutils2.MessageType, severity ext_debugutils2.MessageSeverity, data *ext_debugutils2.CallbackData) bool {
	log.Printf("[%s %s] - %s", severity, msgType, data.Message)
	return false
}

func main() {
	defAlloc := &cgoalloc.DefaultAllocator{}
	lowTier, err := cgoalloc.CreateFixedBlockAllocator(defAlloc, 64*1024, 64, 8)
	if err != nil {
		log.Fatalln(err)
	}

	highTier, err := cgoalloc.CreateFixedBlockAllocator(defAlloc, 4096*1024, 4096, 8)
	if err != nil {
		log.Fatalln(err)
	}

	alloc := cgoalloc.CreateFallbackAllocator(highTier, defAlloc)
	alloc = cgoalloc.CreateFallbackAllocator(lowTier, alloc)

	app := &HelloTriangleApplication{
		allocator: alloc,
	}

	err = app.Run()
	if err != nil {
		log.Fatalln(err)
	}
}