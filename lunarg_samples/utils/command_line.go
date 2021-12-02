package utils

import (
	"fmt"
	"os"
)

func (i *SampleInfo) ProcessCommandLineArgs() error {
	args := os.Args[1:]

	for _, arg := range args {
		if arg == "--save-images" {
			i.SaveImages = true
		} else if arg == "--help" || arg == "-h" {
			fmt.Println("\nOther options")
			fmt.Println("\t--save-images")
			fmt.Println("\t\tSave tests images as ppm files in current working directory")
			os.Exit(0)
			return nil
		} else {
			fmt.Printf("\nUnrecognized option: %s\n", arg)
			fmt.Println("\nUse --help or -h for option list.")
			os.Exit(0)
			return nil
		}
	}

	return nil
}
