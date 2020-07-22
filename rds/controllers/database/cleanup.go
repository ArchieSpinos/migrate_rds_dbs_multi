package database

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/service/rds"
	log "github.com/sirupsen/logrus"
	"github.com/ArchieSpinos/migrate_rds_dbs_multi/rds/dbs"
	"github.com/ArchieSpinos/migrate_rds_dbs_multi/rds/persist"
	"github.com/ArchieSpinos/migrate_rds_dbs_multi/rds/services"
)

func PromoteSlave(a dbs.Access) {
	var (
		pathGlobal               = "/tmp/" + a.SourceHost + "/"
		describeDBInstanceOutput = &rds.DescribeDBInstancesOutput{}
		serviceDBsDest           = &[]string{}
	)

	if err := persist.Load(pathGlobal+"describeInstance", describeDBInstanceOutput); err != nil {
		log.Fatal(err)
	}

	if err := persist.Load(pathGlobal+"serviceDBsDest", &serviceDBsDest); err != nil {
		log.Fatal(err)
	}

	serviceDBsDestString := services.SlicePointerToSlice(serviceDBsDest)

	if err := services.PromoteSlave(a); err != nil {
		log.Fatal(err)
	}

	if err := services.RDSDeleteInstanceCluster(*a.AWSSession, *describeDBInstanceOutput); err != nil {
		log.Infof(err.Error())
	}

	if err := services.CleanUpDBs(a, serviceDBsDestString); err != nil {
		log.Fatal(err)
	}

	if err := persist.DeletePath(pathGlobal); err != nil {
		log.Fatal(err)
	}

	log.Infof(fmt.Sprintf("Cleanup has been complete. %s database has been dropped from %s and %s temp RDS cluster instance has been deleted", a.SourceDBName, a.SourceClusterID, aws.StringValue(describeDBInstanceOutput.DBInstances[0].DBInstanceIdentifier)))
}
