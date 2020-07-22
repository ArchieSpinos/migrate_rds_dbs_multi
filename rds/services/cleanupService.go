package services

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/ArchieSpinos/rgctl/rds/awsclient"
	"github.com/ArchieSpinos/rgctl/rds/dbs"
)

// SlicePointerToSlice accepts a pointer to slice and returns a slice
// with same data.
func SlicePointerToSlice(input *[]string) []string {
	output := append([]string{}, *input...)
	return output
}

// PromoteSlave promotes slave mysql node to master and stops replication.
func PromoteSlave(access dbs.Access) error {
	var (
		stopReplication = "CALL mysql.rds_stop_replication;"
		resetMaster     = "CALL mysql.rds_reset_external_master;"
	)
	var queries = []string{
		stopReplication,
		resetMaster,
	}
	for _, query := range queries {
		result := &dbs.QueryResult{}
		if err := result.MultiColumn(access.DBTarget, query); err != nil {
			return err
		}
	}
	return nil
}

// RDSDeleteInstanceCluster deletes temporary RDS instance and cluster used to dump databases.
func RDSDeleteInstanceCluster(session awsclient.AWSSession, input rds.DescribeDBInstancesOutput) error {
	var (
		deleteInstanceInput = rds.DeleteDBInstanceInput{
			DBInstanceIdentifier: input.DBInstances[0].DBInstanceIdentifier,
			SkipFinalSnapshot:    aws.Bool(true),
		}
		deleteClusterInput = rds.DeleteDBClusterInput{
			DBClusterIdentifier: input.DBInstances[0].DBClusterIdentifier,
			SkipFinalSnapshot:   aws.Bool(true),
		}
	)

	if _, err := session.DeleteDBInstance(&deleteInstanceInput); err != nil {
		return fmt.Errorf(fmt.Sprintf("failed to delete DB instance: %s. Did you remove it manually?",
			err.Error()))
	}

	if _, err := session.DeleteDBCluster(&deleteClusterInput); err != nil {
		return fmt.Errorf(fmt.Sprintf("failed to delete DB cluster: %s. Did you remove it manually?",
			err.Error()))
	}

	return nil
}

// CleanUpDBs drops databases from target cluster which were created as part of
// transactional replication (since RDS does not allow to choose databases when
// creating replication slaves). It also drops migrated database from source cluster
// and resets binlog retention to NULL from source cluster.
func CleanUpDBs(access dbs.Access, serviceDBsDest []string) error {
	var (
		listQuery              = "show databases;"
		resetLogRetentionQuery = "CALL mysql.rds_set_configuration('binlog retention hours', NULL);"
		dropDBQuery            string
		allDestDBs             = dbs.QueryResult{}
		dropMigratedDB         = dbs.QueryResult{}
		resetLog               = dbs.QueryResult{}
		systemsDBs             = []string{"mysql", "performance_schema", "information_schema", "sys", "tmp"}
		tobeRemovedDBs         []string
	)
	serviceDBsDest = append(serviceDBsDest, access.SourceDBName)
	for _, v := range systemsDBs {
		serviceDBsDest = append(serviceDBsDest, v)
	}

	if err := allDestDBs.MultiColumn(access.DBTarget, listQuery); err != nil {
		return err
	}

	for _, v := range allDestDBs {
		for i, k := range serviceDBsDest {
			if v == k {
				break
			} else if (v != k) && (i < len(serviceDBsDest)-1) {
				continue
			} else {
				tobeRemovedDBs = append(tobeRemovedDBs, v)
			}
		}
	}

	for _, db := range tobeRemovedDBs {
		result := &dbs.QueryResult{}
		dropDBQuery = "drop database " + db + ";"
		if err := result.MultiColumn(access.DBTarget, dropDBQuery); err != nil {
			return err
		}
	}

	if err := dropMigratedDB.MultiColumn(access.DBSource, "drop database "+access.SourceDBName+";"); err != nil {
		return err
	}

	if err := resetLog.MultiColumn(access.DBSource, resetLogRetentionQuery); err != nil {
		return err
	}
	return nil
}
