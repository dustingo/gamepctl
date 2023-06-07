/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"path/filepath"
	"strings"

	"github.com/go-kit/log"

	"github.com/spf13/cobra"
)

var (
	cmdFile string
)
var logger log.Logger

// applyCmd represents the apply command
var applyCmd = &cobra.Command{
	Use:   "apply (-f FILENAME)",
	Short: "Apply a server configuration by file name",
}

func init() {
	rootCmd.AddCommand(applyCmd)
	applyCmd.PersistentFlags().StringVarP(&cmdFile, "cmdConfig", "f", "", "A file of commands[yaml]")
	applyCmd.MarkFlagRequired("cmdConfig")
	// applyCmd.MarkFlagsMutuallyExclusive("start", "stop")
}

// verify start 的配置文件必须为start开头，stop的配置文件必须为stop开头
func verify(cmdConfig string, anticipate string) bool {
	_, file := filepath.Split(cmdConfig)
	if strings.HasPrefix(strings.ToLower(file), anticipate) {
		return true
	} else {
		return false
	}
}
