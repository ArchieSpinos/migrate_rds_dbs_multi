package cmd

import (
	"fmt"

	"github.com/ArchieSpinos/rgctl/rds/awsclient"
	"github.com/ArchieSpinos/rgctl/rds/controllers/database"
	"github.com/ArchieSpinos/rgctl/rds/dbs"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var databasePromoteCmd = &cobra.Command{
	Use:   "promote",
	Short: "Promotes slave RDS cluster to master",
	Long:  `Calling this command will break the replication, promote the slave to master, and drop the migrated database from source cluster (!so switching application back to source cluster is not an option). Switchover for the application, regarding mysql endpoint, must take place right after this call as described in the manual runbook.`,

	Run: databasePromote,
}

func init() {
	rdsDatabaseCmd.AddCommand(databasePromoteCmd)
}

func databasePromote(cmd *cobra.Command, args []string) {

	var functions []func()
	initViper := initViper()
	clustersSet := getClustersSet(initViper)

	for _, m := range clustersSet {
		newViper := viper.Sub(m)
		var (
			sourceUser      = newViper.GetString("sourceUser")
			sourcePassword  = newViper.GetString("sourcePassword")
			sourceHost      = newViper.GetString("sourceHost")
			sourceDBName    = newViper.GetString("sourceDBName")
			sourceClusterID = newViper.GetString("sourceClusterID")
			targetUser      = newViper.GetString("targetUser")
			targetPassword  = newViper.GetString("targetPassword")
			targetHost      = newViper.GetString("targetHost")
			awsRegion       = newViper.GetString("awsRegion")
			awsProfile      = newViper.GetString("awsProfile")
			replicaUserPass = newViper.GetString("replicaUserPass")
		)

		sourceDataSourceName := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s",
			sourceUser,
			sourcePassword,
			sourceHost,
			"mysql",
		)

		targetDataSourceName := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s",
			targetUser,
			targetPassword,
			targetHost,
			"mysql",
		)

		sourceMysqlCon, err := dbs.SourceInitConnection(sourceDataSourceName)
		if err != nil {
			log.Fatalf("failed to create DB connection to host %s: %s", sourceHost, err.Error())
		}

		targetMysqlCon, err := dbs.SourceInitConnection(targetDataSourceName)
		if err != nil {
			log.Fatalf("failed to create DB connection to host %s: %s", targetHost, err.Error())
		}

		awsRDSClient, err := awsclient.CreateSession(awsRegion, awsProfile)
		if err != nil {
			log.Fatalf("failed to create AWS RDS client: %s", err.Error())
		}

		access := dbs.Access{
			DBSource:        sourceMysqlCon,
			DBTarget:        targetMysqlCon,
			AWSSession:      awsRDSClient,
			SourceUser:      sourceUser,
			SourcePassword:  sourcePassword,
			SourceHost:      sourceHost,
			TargetUser:      targetUser,
			TargetPassword:  targetPassword,
			TargetHost:      targetHost,
			SourceDBName:    sourceDBName,
			ReplicaUserPass: replicaUserPass,
			SourceClusterID: sourceClusterID,
		}

		functions = append(functions, func() { database.PromoteSlave(access) })
	}
	parallelize(functions)
}
