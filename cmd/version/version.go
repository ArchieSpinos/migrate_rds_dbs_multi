package version

import (
	"fmt"

	"github.com/ArchieSpinos/migrate_rds_dbs_multi/info"
	"github.com/spf13/cobra"
)

var shortPrint bool

var header = `
                  __   __ 
 .----.-----.----|  |_|  |
 |   _|  _  |  __|   _|  |
 |__| |___  |____|____|__|
      |_____|             							 
`

// BaseCmd represents the base Version command when called without any subcommands
var BaseCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints out the version of rgctl.",
	Long:  `Prints out the version of rgctl.`,
	Run:   printVersion,
}

func init() {
	BaseCmd.PersistentFlags().BoolVarP(&shortPrint, "short", "s", false, "Print just the version number.")
}

func printVersion(cmd *cobra.Command, args []string) {
	if shortPrint {
		fmt.Println(info.Version)

	} else {
		fmt.Println(header)
		fmt.Printf("\nrgctl version:%s commit:%s branch:%s date:%s\n", info.Version, info.GitCommit, info.GitBranch, info.BuildDate)
	}
}
