package database

import (
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/ArchieSpinos/migrate_rds_dbs_multi/rds/dbs"
	"github.com/ArchieSpinos/migrate_rds_dbs_multi/rds/services"
)

func SecondsBehindMaster(t dbs.TargetMySQL) {
	result, err := services.CheckSlaveStatus(t)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Transactional replication for %s has been completed with status:\n%v", t.TargetHost, strings.Join(result, "\n"))
}
