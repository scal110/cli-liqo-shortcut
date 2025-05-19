package cmd

import (
	"fmt"
	"os"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "liqoshortcut",
	Short: "a cli command used to manage shortcuts between two foreign clusters",
	Run: func(cmd *cobra.Command, args []string) {
		_ =cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
