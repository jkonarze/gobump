package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	path    string
	version string
)

func Execute() {
	cmdBump.PersistentFlags().StringVarP(&path, "path", "p", "", "path directory of your repos")
	cmdBump.PersistentFlags().StringVarP(&version, "version", "v", "1.14", "desire go version")

	var rootCmd = &cobra.Command{Use: "gobump"}
	rootCmd.AddCommand(cmdBump)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
