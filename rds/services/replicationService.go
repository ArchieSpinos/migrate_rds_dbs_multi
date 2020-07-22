package services

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/ArchieSpinos/migrate_rds_dbs_multi/rds/awsclient"
	"github.com/ArchieSpinos/migrate_rds_dbs_multi/rds/dbs"
)

var (
	systemDBs = []string{"mysql", "performance_schema", "information_schema", "sys", "tmp"}
)

// BootstrapReplication boostraps transactional replication by creating
// replication user and setting binlog retention on source cluster.
func BootstrapReplication(replicaUserPass string, dbSource *dbs.DB) error {
	var (
		dropReplicaUser = "DROP USER IF EXISTS 'repl_user'@'%';"
		replicaUser     = "CREATE USER 'repl_user'@'%' IDENTIFIED BY '" + replicaUserPass + "';"
		grantUser       = "GRANT REPLICATION CLIENT, REPLICATION SLAVE ON *.* TO 'repl_user'@'%';"
		setBinLog       = "CALL mysql.rds_set_configuration('binlog retention hours', 144);"
	)
	var queries = []string{
		dropReplicaUser,
		replicaUser,
		grantUser,
		setBinLog,
	}
	for _, query := range queries {
		result := &dbs.QueryResult{}
		if err := result.MultiColumn(dbSource, query); err != nil {
			return err
		}
	}
	return nil
}

// SetupReplication bootstraps transactional replication at target cluster.
func SetupReplication(sourceHost string, replicaUserPass string, dbTarget *dbs.DB, binLogFile string, binLogPos string) error {
	var (
		setMaster        = "CALL mysql.rds_set_external_master ('" + sourceHost + "', 3306,'repl_user', '" + replicaUserPass + "', '" + binLogFile + "', " + binLogPos + ", 0);"
		startReplication = "CALL mysql.rds_start_replication;"
	)
	var queries = []string{
		setMaster,
		startReplication,
	}
	for _, query := range queries {
		result := &dbs.QueryResult{}
		if err := result.MultiColumn(dbTarget, query); err != nil {
			return err
		}
	}
	return nil
}

func getServiceOnlyDBs(allDBs []string) (userDBs []string) {
	for _, v := range allDBs {
		for ks, vs := range systemDBs {
			if v == vs {
				break
			} else if ks < len(systemDBs)-1 {
				continue
			} else {
				userDBs = append(userDBs, v)
			}
		}
	}
	return userDBs
}

// PreFlightCheck checks that none of the source cluster databases exist in the target
// because that would cause the transactional replication to override the target ones.
func PreFlightCheck(dbSource *dbs.DB, dbTarget *dbs.DB, pathGlobal string) (serviceDBsSource []string, serviceDBsDest []string, err error) {

	var (
		listQuery = "show databases;"
		sourceDBs = dbs.QueryResult{}
		destDBs   = dbs.QueryResult{}
		result    []string
	)

	if _, err := os.Stat(pathGlobal); err == nil {
		return nil, nil, fmt.Errorf(fmt.Sprintf("The dump path %s already exists. You need to delete it first.", pathGlobal))
	}

	if err := sourceDBs.MultiColumn(dbSource, listQuery); err != nil {
		return nil, nil, err
	}
	if err := destDBs.MultiColumn(dbTarget, listQuery); err != nil {
		return nil, nil, err
	}

	serviceDBsSource = getServiceOnlyDBs(sourceDBs)
	serviceDBsDest = getServiceOnlyDBs(destDBs)

	for _, sourceV := range serviceDBsSource {
		for _, destV := range serviceDBsDest {
			if sourceV == destV {
				result = append(result, sourceV)
			}
		}
	}
	if len(result) > 0 {
		return nil, nil, fmt.Errorf(fmt.Sprintf("The following source host databases exist in destination: %v.\n RDS transactional replication will migrate all databases so those existing in destination will be overwritten.\n Cannot continue.", result))
	}
	return serviceDBsSource, serviceDBsDest, nil
}

type RDSDescribe struct {
	Client          rdsiface.RDSAPI
	SourceDBName    string
	SourceClusterID string
}

// Cluster is a representation of the RDS cluster
type Cluster struct {
	DBClusterIdentifier         *string
	DBClusterParameterGroupName *string
	DBSubnetGroupName           *string
	SourceDBClusterIdentifier   *string
	VpcSecurityGroupIds         []*string
	UseLatestRestorableTime     *bool
	RestoreType                 *string
}

// RDSDescribeSourceCluster retrieves information about the source db RDS cluster and returns
// a Cluster that will be used to clone source db cluster to temp cluster from which dumps
// and binlog file and position will be aquired
func (r RDSDescribe) RDSDescribeSourceCluster() (*Cluster, error) {
	var (
		clusterInput = rds.DescribeDBClustersInput{
			DBClusterIdentifier: &r.SourceClusterID,
		}
		DBClusterIdentifier     = "migrate-temp-" + r.SourceClusterID + "-" + r.SourceDBName
		VpcSecurityGroupIdsList []*string
		LatestRestorableTime    = true
		restoreType             = "copy-on-write"
	)
	DBClusterOutput, err := r.Client.DescribeDBClusters(&clusterInput)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("failed to describe RDS Cluster: %s",
			err.Error()))
	}

	for _, element := range DBClusterOutput.DBClusters[0].VpcSecurityGroups {
		VpcSecurityGroupIdsList = append(VpcSecurityGroupIdsList, element.VpcSecurityGroupId)
	}

	cluster := &Cluster{
		DBClusterIdentifier:         &DBClusterIdentifier,
		DBClusterParameterGroupName: DBClusterOutput.DBClusters[0].DBClusterParameterGroup,
		DBSubnetGroupName:           DBClusterOutput.DBClusters[0].DBSubnetGroup,
		SourceDBClusterIdentifier:   &r.SourceClusterID,
		VpcSecurityGroupIds:         VpcSecurityGroupIdsList,
		UseLatestRestorableTime:     &LatestRestorableTime,
		RestoreType:                 &restoreType,
	}

	return cluster, nil
}

// RDSWaitForAddress waits for RDS to assign fqdn to newly created instance. As this can take
// some seconds after instance has been created we need to check before retrieving
func RDSWaitForAddress(rdsClient awsclient.AWSSession, instance *rds.CreateDBInstanceOutput) (*rds.DescribeDBInstancesOutput, error) {
	var (
		instanceInput = rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: instance.DBInstance.DBInstanceIdentifier,
		}
		tries            = 0
		describeInstance *rds.DescribeDBInstancesOutput
	)

	for tries < 36 {
		describeInstance, err := rdsClient.DescribeDBInstances(&instanceInput)
		if err != nil {
			return nil, fmt.Errorf(fmt.Sprintf("failed to describe RDS instance: %s",
				err.Error()))
		} else if describeInstance.DBInstances[0].Endpoint != nil {
			return describeInstance, nil
		} else if tries < 36 {
			time.Sleep(5 * time.Second)
			tries++
		} else {
			return nil, fmt.Errorf(fmt.Sprintf("failed to retrieve RDS instance fqdn"))
		}
	}
	return describeInstance, nil
}

// RDSRestoreCluster performes point-in-time, copy-on-write cluster restore of the source cluster hosting
// the db to be migrated, in order to retrieve dumps and binlog file location.
func RDSRestoreCluster(rdsClient awsclient.AWSSession, input *Cluster, sourceDBName string) (*rds.CreateDBInstanceInput, error) {
	var (
		DBInstanceClassInput      = "db.r4.large"
		DBInstanceIdentifierInput = "migrate-temp-instance-" + aws.StringValue(input.SourceDBClusterIdentifier) + "-" + sourceDBName
		DBEngine                  = "aurora-mysql"
	)

	restoreDBClusterInput := rds.RestoreDBClusterToPointInTimeInput{
		DBClusterIdentifier:         input.DBClusterIdentifier,
		DBClusterParameterGroupName: input.DBClusterParameterGroupName,
		DBSubnetGroupName:           input.DBSubnetGroupName,
		SourceDBClusterIdentifier:   input.SourceDBClusterIdentifier,
		VpcSecurityGroupIds:         input.VpcSecurityGroupIds,
		UseLatestRestorableTime:     input.UseLatestRestorableTime,
		RestoreType:                 input.RestoreType,
	}

	DBClusterOutput, err := rdsClient.RestoreDBClusterToPointInTime(&restoreDBClusterInput)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("failed to restore RDS Cluster: %s",
			err.Error()))
	}

	createDBInstanceInput := rds.CreateDBInstanceInput{
		DBClusterIdentifier:  DBClusterOutput.DBCluster.DBClusterIdentifier,
		DBInstanceClass:      &DBInstanceClassInput,
		DBInstanceIdentifier: &DBInstanceIdentifierInput,
		Engine:               &DBEngine,
	}

	return &createDBInstanceInput, nil
}

// RDSCreateInstance creates an RDS DB instance in the cluster that RDSRestoreCluster func created.
func RDSCreateInstance(rdsClient awsclient.AWSSession, input *rds.CreateDBInstanceInput) (*rds.CreateDBInstanceOutput, error) {
	DBInstanceOutput, err := rdsClient.CreateDBInstance(input)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("failed to create DB instance: %s",
			err.Error()))
	}
	return DBInstanceOutput, nil
}

// RDSWaitUntilInstanceAvailable checks when DB instance that RDSCreateInstance created is ready for connections.
func RDSWaitUntilInstanceAvailable(rdsClient awsclient.AWSSession, dbInstanceOutput *rds.CreateDBInstanceOutput) error {
	const ctxDuration int = 7200
	var (
		input = rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: dbInstanceOutput.DBInstance.DBInstanceIdentifier,
		}
	)
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(ctxDuration)*time.Second)
	if err := rdsClient.WaitUntilDBInstanceAvailableWithContext(ctx, &input); err != nil {
		return fmt.Errorf(fmt.Sprintf("RDS instance did not become available in a timely manner: %s",
			err.Error()))
	}
	return nil
}

// RDSDescribeEvents pulls RDS DB Instance logs in order to find binlog file location and position.
func RDSDescribeEvents(rdsClient awsclient.AWSSession, instance *rds.CreateDBInstanceOutput) (binLogFile *string, binLogPos *string, err error) {
	var (
		tries            = 0
		sourceType       = "db-instance"
		duration   int64 = 7200
		input            = rds.DescribeEventsInput{
			SourceIdentifier: instance.DBInstance.DBInstanceIdentifier,
			SourceType:       &sourceType,
			Duration:         &duration,
		}
	)
	for tries < 720 {
		events, describeErr := rdsClient.DescribeEvents(&input)
		if describeErr != nil {
			return nil, nil, fmt.Errorf(fmt.Sprintf("failed to describe RDS instance events: %s",
				describeErr.Error()))
		}
		for _, v := range events.Events {
			strMessage := aws.StringValue(v.Message)
			if strings.Contains(strMessage, "Binlog position from crash recovery") {
				s := strings.Fields(strMessage)
				return &s[len(s)-2], &s[len(s)-1], nil
			}
		}
		if tries < 720 {
			time.Sleep(5 * time.Second)
			tries++
		} else {
			return nil, nil, fmt.Errorf(fmt.Sprintf("failed to retrieve binlog position and file"))
		}
	}
	return
}
