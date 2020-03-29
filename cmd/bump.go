package cmd

import (
	"github.com/jkonarze/gobump/internal"
	"github.com/spf13/cobra"
)

var cmdBump = &cobra.Command{
	Use:   "bump [path]",
	Short: "Bump version of go for project",
	Long: `An easy way to update the go lang version for the project in the given path`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		svc := internal.NewWorker(args[0], version)
		svc.Init()
	},
}
