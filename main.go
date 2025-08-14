package main

import (
	"fmt"
	"os"
	"github.com/rs/zerolog/log"
	"sohot/e"
	"sohot/watch"
	"sort"

	// "sohot/watch"
	"github.com/manifoldco/promptui"
)

//go:generate go install .
func main() {
	var key string
	
	// Check if command line arguments are provided
	if len(os.Args) > 1 {
		key = os.Args[1]
		log.Info().Str("profile", key).Msg("Using command line specified profile")
	} else {
		// No arguments provided, show interactive selection interface
		items := make([]string, 0, len(e.V.Run))
		for s, run := range e.V.Run {
			if run.Only {
				s += "#Run only mode"
			}
			items = append(items, s)
		}
		sort.Strings(items)
		prompt := promptui.Select{
			Label: "Select profile",
			Items: items,
		}
		_, result, err := prompt.Run()

		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}
		key = extractKey(result)
	}
	
	// Verify if the profile exists
	run, ok := e.V.Run[key]
	if !ok {
		log.Fatal().Str("profile", key).Msg("Profile not found")
	}
	
	watch.Watching(run)
	watch.Building(run)
}
