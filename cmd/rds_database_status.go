package cmd

import (
	"github.com/ArchieSpinos/rgctl/rds/controllers/database"
	"github.com/ArchieSpinos/rgctl/rds/dbs"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var databaseStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Checks the database replication status",
	Long:  `After cloning the cluster and depending on how long the dump/restore took, the master might have moved on substantially from the slave, and the later needs to catch up (replay all transactional log from master). Call this endpoint as many times as needed to check on the replication status.`,

	Run: databaseStatus,
}

func init() {
	rdsDatabaseCmd.AddCommand(databaseStatusCmd)
}

func databaseStatus(cmd *cobra.Command, args []string) {

	var functions []func()
	initViper := initViper()
	clustersSet := getClustersSet(initViper)

	for _, m := range clustersSet {
		newViper := viper.Sub(m)
		var (
			targetUser     = newViper.GetString("targetUser")
			targetPassword = newViper.GetString("targetPassword")
			targetHost     = newViper.GetString("targetHost")
		)

		targetMySQL := dbs.TargetMySQL{
			TargetHost:     targetHost,
			TargetPassword: targetPassword,
			TargetUser:     targetUser,
		}

		functions = append(functions, func() { database.SecondsBehindMaster(targetMySQL) })
	}
	parallelize(functions)
}
