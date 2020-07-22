# devops-migrate-mysql-db-multi
This tool perfoms the steps of [AWS Replication between Aurora clusters documentation](https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/AuroraMySQL.Replication.MySQL.html) but for multiple Aurora clusters simultaneously.

Skim through the guide first to understand the process and actors involved.

### :white_check_mark: Prerequisites

- mysql 5.7 client 
- aws profile with RDS permissions:
    - RestoreCluster
    - DescribeClusters
    - CreateDBInstance
    - DescribeEvents
    - DeleteDBInstance
    - DeleteDBCluster

:rocket: The RDS command workflow comprises of three commands that can be seen by running: 

`rgctl rds database --help`

The three commands must be called in order of below appearance:

### migrate

`rgctl rds database migrate`

Takes care of all steps to setup replication between source and target clusters. ie: Set binlog retention, clone source cluster, dump/restore all databases from clone, bootstrap replication.

The tool will try to run for all configured clusters (see later in configuration) in parallel and log info for each one as it progresses through the procedure.

This command needs to persist some return objects to your local disk in `/tmp/source-Host+migrated-DB-name/` to be used in cleanup command later, so do not remove this directory until you have finished with the migration process.
Also the partition mounted in `/tmp/` must be big enough to hold the dumps from all the clusters that you plan to migrate the database for.

Once you get a success exit code move on to the next command.

### :exclamation: Gotchas
Dumping and restoring databases can take a long time depending on the size, and if during this time the command exits, for whatever reason, there is no way to start at where it left off. A manual cleanup, deleting temporary cluster, and beginning from the start is the only way in this version.

### Replication status

`rgctl rds database status`

After cloning the cluster and depending on how long the dump/restore took, the master might have moved on substantially from the slave, and the later needs to catch up (replay all transactional log from master).
Run this command as many times as needed to check on the replication status. 
At this stage at least in a sandbox, a switchover of the application to the new mysql endpoint can take place, and readonly tests (because cluster is still a slave) can be executed. 

Once you get a a success exit code it's time to move on to the final command, but before you do read carefully..

### :sos: Promote slave (point of no return)

`rgctl rds database promote` 

Running this command will break the replication, promote the slave to master, and drop the migrated database from source cluster (!so switching application back to source cluster is not an option). Switchover for the application, regarding mysql endpoint, must take place right after this command.

### Configuration

All three commands expect the below sample JSON configuration file in `./rds/config/clusters.json`. Copy a section for all clusters that you need to migrate a database for and name it with any of `[CL1, CL2, CL3, CL4, CL5, CL6, CL7]` for the top level key.

```
{
    "GR": {
        "sourceUser": "root",
        "sourcePassword": "howaboutthispass",
        "sourceHost":"dev-migrate-source-instance-1.cmnsml8q1eeo.eu-west-1.rds.amazonaws.com",
        "sourceClusterID":"dev-migrate-source",
        "sourceDBName":"source2",
        "awsRegion":"eu-west-1",
        "awsProfile":"a.spinos",
        "replicaUserPass":"Edeur2NBB2sQLdq4JdqF5fTdFTw98S",
        "targetUser":"root",
        "targetPassword":"killerpass",
        "targetHost":"dev-migrate-target-instance-1.cmnsml8q1eeo.eu-west-1.rds.amazonaws.com"
    },
    "CO": {
        "sourceUser": "admin",
        "sourcePassword": "MytestPassword!",
        "sourceHost":"dev-migrate-source.cluster-cw4i1mpvfsgk.us-west-1.rds.amazonaws.com",
        "sourceClusterID":"dev-migrate-source",
        "sourceDBName":"sourceDB1",
        "awsRegion":"us-west-1",
        "awsProfile":"a.spinos",
        "replicaUserPass":"EdeurBfdsHJG8qF5fTdFTw98S",
        "targetUser":"administrator",
        "targetPassword":"AnotherAwesomePassword",
        "targetHost":"dev-migrate-target-instance-1.cw4i1mpvfsgk.us-west-1.rds.amazonaws.com"
    }
}
```

### :no_entry: Steps not involved

Migrating user accounts. User accounts should best be created again in the target cluster.
Service accounts for the migrated database will be added in the next iteration.
