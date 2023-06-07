/*
Copyright © 2022 zack <514838728@qq.com>

*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// var serverConfig string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:           "gamepctl",
	Short:         "A application used for game process control",
	SilenceErrors: false,
	SilenceUsage:  true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// func init() {
// 	rootCmd.Flags().StringVarP(&serverConfig, "config", "c", "server.toml", "game servers config file[toml]")
// }
// func InitServerConfig() string {
// 	if serverConfig == "" {
// 		ex, err := os.Executable()
// 		if err != nil {
// 			panic(err)
// 		}
// 		serverConfig = fmt.Sprintf("%s/%s", filepath.Dir(ex), serverConfig)
// 		return serverConfig
// 	} else {
// 		return serverConfig
// 	}
// }
