package main

import (
	"github.com/CannibalVox/VKng"
	"log"
)

func instance_creation() {
	alloc := &VKng.DefaultAllocator{}
	i, err := (&VKng.InstanceBuilder{}).
		ApplicationName("Hello Triangle").
		ApplicationVersion(1, 0, 0).
		EngineName("No Engine").
		EngineVersion(1, 0, 0).
		Build(alloc)
	if err != nil {
		log.Fatalln(err)
	}
	defer i.Destroy()
}
