package cmd

import (
	"fmt"
	"sync"

	"github.com/ArchieSpinos/rgctl/rds/controllers/database"

	"github.com/ArchieSpinos/rgctl/rds/awsclient"
	"github.com/ArchieSpinos/rgctl/rds/dbs"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var databaseMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrates a database across AWS clusters",
	Long:  `Takes care of all steps to setup replication between source and target clusters. ie: Set binlog retention, clone source cluster, dump/restore all databases from clone, bootstrap replication.`,

	Run: databaseMigrate,
}

func init() {
	rdsDatabaseCmd.AddCommand(databaseMigrateCmd)
}

func parallelize(functions []func()) {
	var waitGroup sync.WaitGroup
	waitGroup.Add(len(functions))

	defer waitGroup.Wait()

	for _, function := range functions {
		go func(rdsOperation func()) {
			defer waitGroup.Done()
			rdsOperation()
		}(function)
	}
}

func initViper() func() {
	return func() {
		func() {
			viper.SetConfigName("clusters")
			viper.SetConfigType("json")
			viper.AddConfigPath("./rds/config/")
			err := viper.ReadInConfig()
			if err != nil {
				log.Fatalf("Fatal error config file: %s \n", err)
			}
		}()
	}
}

func getClustersSet(viperConfig func()) (clustersSet []string) {
	viperConfig()
	clustersAll := []string{"CL1", "CL2", "CL3", "CL4", "CL5", "CL6", "CL7"}
	for _, v := range clustersAll {
		if viper.IsSet(v) {
			clustersSet = append(clustersSet, v)
		} else {
			continue
		}
	}
	return clustersSet
}

func databaseMigrate(cmd *cobra.Command, args []string) {

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
			log.Fatalf("failed to create DB connection to cluster %s: %s", sourceHost, err.Error())
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

		functions = append(functions, func() { database.SetupReplication(access) })
	}
	parallelize(functions)
}
