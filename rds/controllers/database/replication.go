package database

import (
	"github.com/aws/aws-sdk-go/aws"
	log "github.com/sirupsen/logrus"
	"github.com/ArchieSpinos/rgctl/rds/dbs"
	"github.com/ArchieSpinos/rgctl/rds/persist"
	"github.com/ArchieSpinos/rgctl/rds/services"
)

func SetupReplication(a dbs.Access) {
	var pathGlobal = "/tmp/" + a.SourceHost + "/"

	serviceDBsSource, serviceDBsDest, err := services.PreFlightCheck(a.DBSource, a.DBTarget, pathGlobal)
	if err != nil {
		log.Fatal(err)
	}

	if err := persist.CreatePath(pathGlobal); err != nil {
		log.Fatal(err)
	}

	if err := persist.Save(pathGlobal, "serviceDBsDest", serviceDBsDest); err != nil {
		log.Fatal(err)
	}

	if err := services.BootstrapReplication(a.ReplicaUserPass, a.DBSource); err != nil {
		log.Fatal(err)
	}

	r := services.RDSDescribe{
		Client:          a.AWSSession,
		SourceDBName:    a.SourceDBName,
		SourceClusterID: a.SourceClusterID,
	}

	dbClusters, err := r.RDSDescribeSourceCluster()
	if err != nil {
		log.Fatal(err)
	}

	createDBInstanceInput, err := services.RDSRestoreCluster(*a.AWSSession, dbClusters, a.SourceDBName)
	if err != nil {
		log.Fatal(err)
	}

	rdsInstance, err := services.RDSCreateInstance(*a.AWSSession, createDBInstanceInput)
	if err != nil {
		log.Fatal(err)
	}

	// // mock CreateDBInstanceOutput
	// var instance = "migrate-temp-instance-source3"
	// var address = "migrate-temp-instance-source3.cmnsml8q1eeo.eu-west-1.rds.amazonaws.com"
	// rdsInstance := &rds.CreateDBInstanceOutput{
	// 	DBInstance: &rds.DBInstance{
	// 		DBInstanceIdentifier: &instance,
	// 		Endpoint: &rds.Endpoint{
	// 			Address: &address,
	// 		},
	// 	},
	// }

	if err := services.RDSWaitUntilInstanceAvailable(*a.AWSSession, rdsInstance); err != nil {
		log.Fatal(err)
	}

	describeInstance, err := services.RDSWaitForAddress(*a.AWSSession, rdsInstance)
	if err != nil {
		log.Fatal(err)
	}

	if err := persist.Save(pathGlobal, "describeInstance", describeInstance); err != nil {
		log.Fatal(err)
	}

	binLogFile, binLogPos, err := services.RDSDescribeEvents(*a.AWSSession, rdsInstance)
	if err != nil {
		log.Fatal(err)
	}

	if err := dbs.MysqlDumpExec(a.SourceUser, a.SourcePassword, aws.StringValue(describeInstance.DBInstances[0].Endpoint.Address), serviceDBsSource, pathGlobal); err != nil {
		log.Fatal(err)
	}

	if err := dbs.MysqlRestore(a.TargetHost, a.TargetUser, a.TargetPassword, pathGlobal); err != nil {
		log.Fatal(err)
	}

	if err := services.SetupReplication(a.SourceHost, a.ReplicaUserPass, a.DBTarget, aws.StringValue(binLogFile), aws.StringValue(binLogPos)); err != nil {
		log.Fatal(err)
	}

	log.Infof("Transactional replication between source: %s and taget: %s has been setup. You now need to monitor with `status` command that `Seconds_Behind_Master` of mysql> show slave status; has reached zero after which you need to coordinate the microservice mysql switchover and call `promote` command", a.SourceHost, a.TargetHost)
}
