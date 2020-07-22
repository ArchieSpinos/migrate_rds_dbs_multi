package dbs

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"

	"github.com/ArchieSpinos/rgctl/rds/persist"
)

func MysqlDumpExec(sourceUser string, sourcePassword string, restoredInstanceDNS string, serviceDBs []string, pathGlobal string) error {
	var (
		out    bytes.Buffer
		stderr bytes.Buffer
	)

	if err := persist.CreatePath(pathGlobal + "sqldumps"); err != nil {
		return err
	}

	for _, serviceDB := range serviceDBs {
		dumpFile := pathGlobal + "sqldumps/" + serviceDB + ".sql"
		cmd := exec.Command("mysqldump", "--databases", serviceDB, "--single-transaction", "--set-gtid-purged=OFF", "--compress", "--order-by-primary", "-r", dumpFile, "-h", restoredInstanceDNS, "-u", sourceUser, "-p"+sourcePassword)
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf(fmt.Sprintf("Error dumping all source host databases: %s", stderr.String()))
		}
	}
	return nil
}

func MysqlRestore(targetHost string, targetUser string, targetPassword string, pathGlobal string) error {
	var (
		out    bytes.Buffer
		stderr bytes.Buffer
	)
	files, err := ioutil.ReadDir(pathGlobal + "/sqldumps")
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("Error listing dump files: %s", err.Error()))
	}

	for _, v := range files {
		execute := "source " + pathGlobal + "/sqldumps/" + v.Name()
		cmd := exec.Command("mysql", "-h", targetHost, "-u", targetUser, "-p"+targetPassword, "-e", execute)
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf(fmt.Sprintf("Error restoring database: %s", stderr.String()))
		}
	}
	return nil
}
