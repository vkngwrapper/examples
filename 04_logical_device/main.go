package main

import (
	"errors"
	"github.com/CannibalVox/VKng"
	"github.com/CannibalVox/VKng/creation"
	"github.com/CannibalVox/VKng/ext_debugutils"
	"github.com/CannibalVox/VKng/objects"
	"github.com/CannibalVox/cgoalloc"
	"github.com/palantir/stacktrace"
	"github.com/veandco/go-sdl2/sdl"
	"log"
)

type HelloTriangleApplication struct {
	allocator cgoalloc.Allocator
	window *sdl.Window

	instance *objects.Instance
	debugMessenger *ext_debugutils.Messenger
	physicalDevice *objects.PhysicalDevice
	logicalDevice *objects.Device
	queue *objects.Queue
}

func (app *HelloTriangleApplication) Run() error {
	err := app.initWindow()
	if err != nil {return err }

	err = app.initVulkan()
	if err != nil { return err }
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

func (app *HelloTriangleApplication) createInstance() error {
	instanceOptions := &creation.InstanceOptions{
		ApplicationName: "Hello Triangle",
		ApplicationVersion: VKng.CreateVersion(1, 0, 0),
		EngineName: "No Engine",
		EngineVersion: VKng.CreateVersion(1, 0, 0),
		VulkanVersion: 		creation.Vulkan1_2,
	}

	// Add extensions
	sdlExtensions := app.window.VulkanGetInstanceExtensions()
	extensions, err := VKng.AvailableExtensions(app.allocator)
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

	instanceOptions.ExtensionNames = append(instanceOptions.ExtensionNames, ext_debugutils.ExtensionName)

	// Add layers
	layers, err := VKng.AvailableLayers(app.allocator)
	if err != nil {
		return err
	}

	_, hasOptimus := layers["VK_LAYER_NV_optimus"]
	if !hasOptimus {
		return errors.New("createInstance: cannot add nvidia optimus layer")
	}
	instanceOptions.LayerNames = append(instanceOptions.LayerNames, "VK_LAYER_NV_optimus")

	_, hasValidation := layers["VK_LAYER_KHRONOS_validation"]
	if !hasValidation {
		return errors.New("createInstance: cannot add khronos validation layer- install LunarG Vulkan SDK")
	}
	instanceOptions.LayerNames = append(instanceOptions.LayerNames, "VK_LAYER_KHRONOS_validation")

	// Add debug messenger
	debugMessengerOptions := &ext_debugutils.Options{
		CaptureSeverities: ext_debugutils.SeverityError|ext_debugutils.SeverityWarning,
		CaptureTypes: ext_debugutils.TypeAll,
		Callback: app.logDebug,
	}
	instanceOptions.Next = debugMessengerOptions

	app.instance, err = objects.CreateInstance(app.allocator, instanceOptions)
	if err != nil {
		return err
	}

	app.debugMessenger, err = ext_debugutils.CreateMessenger(app.allocator, app.instance, debugMessengerOptions)
	if err != nil {
		return err
	}

	return nil
}

func (app *HelloTriangleApplication) initVulkan() error {
	err := app.createInstance()
	if err != nil {
		return err
	}

	err = app.pickPhysicalDevice()
	if err != nil {
		return err
	}

	return app.createLogicalDevice()
}

func (app *HelloTriangleApplication) cleanup() {
	if app.logicalDevice != nil {
		app.logicalDevice.Destroy()
	}

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

func (app *HelloTriangleApplication) logDebug(msgType ext_debugutils.MessageType, severity ext_debugutils.MessageSeverity, data *ext_debugutils.CallbackData) bool {
	log.Printf("[%s %s] - %s", severity, msgType, data.Message)
	return false
}

func (app *HelloTriangleApplication) mainLoop() error {
	for true {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch event.(type) {
			case *sdl.QuitEvent:
				return nil
			}
		}
	}

	return nil
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
