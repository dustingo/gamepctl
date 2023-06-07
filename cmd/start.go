/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"errors"
	"fmt"
	"gamepctl/config"
	"gamepctl/control"
	"os"
	"strings"
	"sync"

	"github.com/go-kit/log"
	"github.com/spf13/cobra"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start game process",
	RunE: func(cmd *cobra.Command, args []string) error {
		switch verify(cmdFile, "start") {
		case true:
			cfg, err := config.LoadCmdConfig(cmdFile)
			if err != nil {
				return err
			}
			if strings.ToLower(cfg.Kind) != "start" {
				return errors.New(fmt.Sprintln("running start operation but kind is ", cfg.Kind))
			}
			var lock sync.Mutex
			logger = log.NewLogfmtLogger(os.Stdout)
			logger = log.With(logger, "caller", log.DefaultCaller)
			gamectl := control.NewControl(logger, cfg, &lock)
			gamectl.Run()
		case false:
			return errors.New(fmt.Sprintln("config file should start with \"start\""))
		}
		return nil
	},
}

func init() {
	applyCmd.AddCommand(startCmd)
}
