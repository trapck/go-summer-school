package cmd

import (
	_ "github.com/jackc/pgx/v4/stdlib" // postgres driver

	"wasfaty.api/pkg/cmd/cobra"
)

//nolint:gochecknoinits
func init() {
	cmdRoot.AddCommand(cmdRun)

	cobra.LoadFlagEnv(cmdRoot.PersistentFlags(), &envFile)
}

var (
	envFile string
	cmdRoot = cobra.CmdRoot(&envFile)
	cmdRun  = cobra.CmdRunService(runService)
)

func Run() error {
	return cmdRoot.Execute()
}
