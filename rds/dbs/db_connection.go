package dbs

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

type DB struct {
	*sql.DB
}

func SourceInitConnection(dataSourceName string) (*DB, error) {

	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		//log(err)
		return nil, err
	}
	return &DB{db}, nil
}
