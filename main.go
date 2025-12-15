package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/qwenode/sohot/boot"
	"github.com/qwenode/sohot/i18n"
	"github.com/qwenode/sohot/version"
	"github.com/qwenode/sohot/watch"

	"github.com/rs/zerolog/log"

	"github.com/manifoldco/promptui"
)

//go:generate go install .
func main() {
	var key string

	// Check if command line arguments are provided
	if len(os.Args) > 1 {
		arg := os.Args[1]

		// Handle version commands
		if arg == "--version" || arg == "-v" || arg == "version" {
			buildInfo := version.GetBuildInfo()
			fmt.Println(buildInfo.String())
			return
		}

		key = arg
		log.Info().Str("profile", key).Msg(i18n.T("main.profile_selected"))
	} else {
		// No arguments provided, show interactive selection interface
		items := make([]string, 0, len(boot.V.Run))
		for s, run := range boot.V.Run {
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
	run, ok := boot.V.Run[key]
	if !ok {
		log.Fatal().Str("profile", key).Msg(i18n.T("main.profile_unknown"))
	}

	buildInfo := version.GetBuildInfo()
	log.Info().Str("version", buildInfo.Version).Str("commit", buildInfo.Commit).Msg(i18n.T("main.initialized"))

	watch.New(run).Start()
}

func extractKey(result string) string {
	if idx := strings.Index(result, "#"); idx != -1 {
		return result[:idx]
	}
	return result
}
