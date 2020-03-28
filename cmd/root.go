package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var (
	path string
)

func Execute() {
	cmdBump.PersistentFlags().StringVarP(&path, "path", "p", "", "path to your repo")

	var rootCmd = &cobra.Command{Use: "gobump"}
	rootCmd.AddCommand(cmdBump)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
