package services

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
)

func (c *Cluster) String() string {
	return fmt.Sprintf("Cluster{DBClusterIdentifier:%v, DBClusterParameterGroupName:%v, DBSubnetGroupName:%v, SourceDBClusterIdentifier:%v, VpcSecurityGroupIds:%v, UseLatestRestorableTime:%v, RestoreType:%v}", aws.StringValue(c.DBClusterIdentifier), aws.StringValue(c.DBClusterParameterGroupName), aws.StringValue(c.DBSubnetGroupName), aws.StringValue(c.SourceDBClusterIdentifier), aws.StringValueSlice(c.VpcSecurityGroupIds), aws.BoolValue(c.UseLatestRestorableTime), aws.StringValue(c.RestoreType))
}

type mockedRDSDescribeCluster struct {
	rdsiface.RDSAPI
	Resp rds.DescribeDBClustersOutput
}

func (m mockedRDSDescribeCluster) DescribeDBClusters(in *rds.DescribeDBClustersInput) (*rds.DescribeDBClustersOutput, error) {
	return &m.Resp, nil
}

func TestRDSDescribeClusterNoError(t *testing.T) {
	cases := []struct {
		Resp     rds.DescribeDBClustersOutput
		Expected *Cluster
	}{
		{ // Case 1, expect Cluster
			Resp: rds.DescribeDBClustersOutput{
				DBClusters: []*rds.DBCluster{
					{
						DBClusterParameterGroup: aws.String("test_dbparamgroup"),
						DBSubnetGroup:           aws.String("test_dbsubnetgroup"),
						VpcSecurityGroups: []*rds.VpcSecurityGroupMembership{
							{VpcSecurityGroupId: aws.String("vpcSGtest1")},
							{VpcSecurityGroupId: aws.String("vpcSGtest2")},
						},
					},
				},
			},
			Expected: &Cluster{
				DBClusterIdentifier:         aws.String("migrate-temp-test_cluster-test_dbname"),
				DBClusterParameterGroupName: aws.String("test_dbparamgroup"),
				DBSubnetGroupName:           aws.String("test_dbsubnetgroup"),
				SourceDBClusterIdentifier:   aws.String("test_cluster"),
				VpcSecurityGroupIds:         []*string{aws.String("vpcSGtest1"), aws.String("vpcSGtest2")},
				UseLatestRestorableTime:     aws.Bool(true),
				RestoreType:                 aws.String("copy-on-write"),
			},
		},
	}

	for i, c := range cases {
		r := RDSDescribe{
			Client:          mockedRDSDescribeCluster{Resp: c.Resp},
			SourceDBName:    "test_dbname",
			SourceClusterID: "test_cluster",
		}
		dbClusterOutput, err := r.RDSDescribeSourceCluster()
		if err != nil {
			t.Fatalf("%d, unexpected error, %v", i, err)
		}
		if !reflect.DeepEqual(dbClusterOutput, c.Expected) {
			fmt.Printf("%d, expected %v, got %v", i, c.Expected, dbClusterOutput)
		}
	}
}
