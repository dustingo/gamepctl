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

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop game process",
	RunE: func(cmd *cobra.Command, args []string) error {
		switch verify(cmdFile, "stop") {
		case true:
			cfg, err := config.LoadCmdConfig(cmdFile)
			if err != nil {
				return err
			}
			if strings.ToLower(cfg.Kind) != "stop" {
				return errors.New(fmt.Sprintln("running stop operation but kind is ", cfg.Kind))
			}
			var lock sync.Mutex
			logger = log.NewLogfmtLogger(os.Stdout)
			logger = log.With(logger, "caller", log.DefaultCaller)
			gamectl := control.NewControl(logger, cfg, &lock)
			gamectl.Run()
		case false:
			return errors.New(fmt.Sprintln("config file should start with \"stop\""))
		}
		return nil
	},
}

func init() {
	applyCmd.AddCommand(stopCmd)

}
