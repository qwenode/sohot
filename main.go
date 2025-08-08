package main

import (
	"fmt"
	"os"
	"github.com/rs/zerolog/log"
	"rr"
	"sohot/e"
	"sohot/watch"
	"sort"

	// "sohot/watch"
	"github.com/manifoldco/promptui"
)

//go:generate go install .
func main() {
	var key string
	
	// 检查是否提供了命令行参数
	if len(os.Args) > 1 {
		key = os.Args[1]
		log.Info().Str("配置", key).Msg("使用命令行指定的配置")
	} else {
		// 没有提供参数，显示交互式选择界面
		items := make([]string, 0, len(e.V.Run))
		for s, run := range e.V.Run {
			if run.Only {
				s += "#仅运行"
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
		key = rr.NewS(result).GetFirst("#").String()
	}
	
	// 验证配置是否存在
	run, ok := e.V.Run[key]
	if !ok {
		log.Fatal().Str("配置", key).Msg("配置不存在")
	}
	
	watch.Watching(run)
	watch.Building(run)
}
