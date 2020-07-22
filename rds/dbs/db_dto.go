package dbs

import "github.com/ArchieSpinos/rgctl/rds/awsclient"

type DbConnection struct {
	Name     string `json:"db_name"`
	User     string `json:"db_user"`
	Password string `json:"db_password"`
	Host     string `json:"db_host"`
}

type QueryResult []string

type Access struct {
	DBSource        *DB
	DBTarget        *DB
	AWSSession      *awsclient.AWSSession
	SourceUser      string
	SourcePassword  string
	SourceHost      string
	TargetUser      string
	TargetPassword  string
	TargetHost      string
	SourceDBName    string
	ReplicaUserPass string
	SourceClusterID string
}

type TargetMySQL struct {
	TargetHost     string
	TargetUser     string
	TargetPassword string
}
