package cmd

import (
	"github.com/spf13/cobra"
)

var rdsDatabaseCmd = &cobra.Command{
	Use:   "database",
	Short: "A list of RDS database operations",
}

func init() {
	rdsCmd.AddCommand(rdsDatabaseCmd)
}
