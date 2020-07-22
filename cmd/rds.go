package cmd

import (
	"github.com/spf13/cobra"
)

var rdsCmd = &cobra.Command{
	Use:   "rds",
	Short: "A list of RDS operations",
}

func init() {
	RootCmd.AddCommand(rdsCmd)
}
