package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	path           string
	version        string
	builderVersion string
	buildImage     string
	runImage       string
)

func Execute() {
	cmdBump.PersistentFlags().StringVarP(&path, "path", "p", "", "path to go repos")
	cmdBump.PersistentFlags().StringVarP(&version, "version", "v", "1.14", "desire go version")
	cmdBump.PersistentFlags().StringVarP(&builderVersion, "builderVersion", "b", "v9.1.0", "desire voi builder version")
	cmdBump.PersistentFlags().StringVarP(&buildImage, "buildImage", "i", "1.17", "desire golang docker build image")
	cmdBump.PersistentFlags().StringVarP(&runImage, "runImage", "r", "3.14", "desire golang docker run image")

	var rootCmd = &cobra.Command{Use: "gobump"}
	rootCmd.AddCommand(cmdBump)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
