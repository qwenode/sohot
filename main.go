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
	if len(os.Args) > 1 {
		arg := os.Args[1]
		if arg == "--version" || arg == "-v" || arg == "version" {
			buildInfo := version.GetBuildInfo()
			fmt.Println(buildInfo.String())
			return
		}
	}

	boot.Load()

	var key string

	if len(os.Args) > 1 {
		key = strings.ToLower(os.Args[1])
		log.Info().Str("profile", key).Msg(i18n.T("main.profile_selected"))
	} else {
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

	run, ok := boot.V.Run[key]
	if !ok {
		log.Fatal().Str("profile", key).Msg(i18n.T("main.profile_unknown"))
	}

	if cwd, err := os.Getwd(); err == nil {
		projectName := cwd[strings.LastIndex(cwd, string(os.PathSeparator))+1:]
		boot.SetConsoleTitle(key + " - " + projectName + " - SoHot")
	} else {
		boot.SetConsoleTitle(key + " - SoHot")
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
