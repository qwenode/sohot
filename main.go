package main

import (
    "fmt"
    "github.com/rs/zerolog/log"
    "rr"
    "sohot/e"
    "sohot/watch"
    "sort"
    
    // "sohot/watch"
    "github.com/manifoldco/promptui"
)

func main() {
	items := make([]string, 0, len(e.V.Run))
	for s, run := range e.V.Run {
		if run.Only {
			s+="#仅运行"
		}
		
		items = append(items, s)
	}
    sort.Strings(items)
	prompt := promptui.Select{
		Label: "选择配置",
		Items: items,
	}
	_, result, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}
    key := rr.NewS(result).GetFirst("#").String()
    run,ok := e.V.Run[key]
    if !ok {
        log.Fatal().Str("配置",key).Msg("不存在")
    }
    watch.Watching(run)
    watch.Building(run)
}
