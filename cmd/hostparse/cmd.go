package hostparse

import (
	"fmt"
	"os"

	"github.com/brian1917/illumioapi/v2"
	"github.com/brian1917/workloader/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// data structure built from the parser.csv
type regex struct {
	Regexdata []regexstruct
}

// regex structure with regex and array of replace regex to build the labels
type regexstruct struct {
	regex   string
	labelcg map[string]string
}

// Input is the data structure the FromCSV function expects
type Input struct {
	PCE                                         illumioapi.PCE
	ImportFile                                  string
	ImportData                                  [][]string
	RemoveValue                                 string
	RolePrefix, AppPrefix, EnvPrefix, LocPrefix string
	Headers                                     map[string]int
	// MatchIndex                                                                               *int
	MatchString                                                                                               string
	Umwl, KeepAllPCEInterfaces, FQDNtoHostname, AllowEnforcementChanges, UpdateWorkloads, UpdatePCE, NoPrompt bool
	ManagedOnly                                                                                               bool
	UnmanagedOnly                                                                                             bool
	IgnoreCase                                                                                                bool
	MaxUpdate, MaxCreate                                                                                      int
	RegexCsv                                                                                                  [][]string
}

// Set up global variables
var parserFile, hostFile, outputFileName string
var debug, noPrompt, updatePCE, allWklds bool
var err error

// input is a global variable bases on wkld-import "input" but with regexcsv object
var input Input

// Init function will handle flags
func init() {
	HostnameCmd.Flags().StringVar(&hostFile, "hostfile", "", "Location of optional CSV file with target hostnames parse. Used instead of getting workloads from the PCE.")
	HostnameCmd.Flags().BoolVar(&allWklds, "all", false, "Parse all PCE workloads no matter what labels are assigned. Individual label flags are ignored if set.")
	HostnameCmd.Flags().StringVar(&outputFileName, "output-file", "", "optionally specify the name of the output file location. default is current location with a timestamped filename.")

	HostnameCmd.Flags().SortFlags = false

}

// HostnameCmd runs the hostname parser
var HostnameCmd = &cobra.Command{
	Use:   "hostparse [parser file csv]",
	Short: "Label workloads by parsing hostnames from provided regex functions.",
	Long: `
Label workloads by parsing hostnames.

An input CSV specifics the regex functions to use to assign labels. An example is below:

+-----------------------------------------------------+------+------+-----------+-----------+
|                        REGEX                        | ROLE | APP  |    ENV    | LOC       |
+-----------------------------------------------------+------+------+-----------+-----------+
| ([A-Za-z]{3})-([4]).*                               |      | ${1} | CERT      |           |
| ([A-Za-z]{3})-([7]).*                               |      | ${1} | DEV       |           |
| ([A-Za-z0-9]*)\.([A-Za-z0-9]*)\.([A-Za-z0-9]*)\.\w+ | ${1} | ${2} |           |           |
| (h)(3)-(\w*)-([sd])(\d+)                            | APP  | ${3} | SITE${5}  | Amazon    |
| (h)(6)-(\w*)-([sd])(\d+)                            | DB   | ${3} | SITE${5}  | Amazon    |
+-----------------------------------------------------+------+------+-----------+-----------+


`,
	Run: func(cmd *cobra.Command, args []string) {

		var err error
		input.PCE, err = utils.GetTargetPCEV2(true)
		if err != nil {
			utils.Logger.Fatalf("Error getting PCE for csv command - %s", err)
		}

		// Set the CSV file
		if len(args) != 1 {
			fmt.Println("Command requires 1 argument for the csv file. See usage help.")
			os.Exit(0)
		}

		// Get the debug value from viper
		input.UpdatePCE = viper.Get("update_pce").(bool)
		input.NoPrompt = viper.Get("no_prompt").(bool)

		// Load the PCE with workloads
		apiResps, err := input.PCE.Load(illumioapi.LoadInput{Workloads: true}, utils.UseMulti())
		utils.LogMultiAPIRespV2(apiResps)
		if err != nil {
			utils.LogError(err.Error())
		}
		var error error
		input.RegexCsv, input.Headers, error = utils.ParseCsvHeaders(args[0])
		if error != nil {
			utils.LogError(err.Error())
		}

		hostnameParser(input)
	},
}
