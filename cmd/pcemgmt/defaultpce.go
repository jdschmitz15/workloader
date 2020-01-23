package pcemgmt

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/brian1917/workloader/utils"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// SetDefaultPCECmd sets the default PCE
var SetDefaultPCECmd = &cobra.Command{
	Use:   "set-default [name of pce]",
	Short: "Changes the default PCE to be used for all commands targeting a single PCE (i.e., do not transfer data between PCEs).",
	Long: `
Changes the default PCE to be used for all commands targeting a single PCE (i.e., do not transfer data between PCEs).`,
	PreRun: func(cmd *cobra.Command, args []string) {
		configFilePath, err = filepath.Abs(viper.ConfigFileUsed())
		if err != nil {
			utils.Log(1, err.Error())
		}
	},
	Run: func(cmd *cobra.Command, args []string) {

		// Set the CSV file
		if len(args) != 1 {
			fmt.Println("Command requires 1 argument for the name of the new deafult PCE. See usage help.")
			os.Exit(0)
		}
		newDefaultPCE := args[0]

		utils.Log(0, "set-default command started")

		// Make sure PCE exists in the YAML file
		if viper.Get(newDefaultPCE+".fqdn") == nil {
			utils.Log(1, fmt.Sprintf("%s PCE does not exist.", newDefaultPCE))
		}

		viper.Set("default_pce_name", newDefaultPCE)
		if err := viper.WriteConfig(); err != nil {
			utils.Log(1, err.Error())
		}

		fmt.Printf("%s is default PCE.\r\n", newDefaultPCE)

		utils.Log(0, "set-default command completed")

	},
}

// GetDefaultPCECmd prints the default PCE
var GetDefaultPCECmd = &cobra.Command{
	Use:   "get-default",
	Short: "Get the default PCE to be used for all commands targeting a single PCE (i.e., do not transfer data between PCEs).",
	Long: `
Get the default PCE to be used for all commands targeting a single PCE (i.e., do not transfer data between PCEs).`,
	PreRun: func(cmd *cobra.Command, args []string) {
		configFilePath, err = filepath.Abs(viper.ConfigFileUsed())
		if err != nil {
			utils.Log(1, err.Error())
		}
	},
	Run: func(cmd *cobra.Command, args []string) {

		utils.Log(0, "get-default command started")

		fmt.Printf("%s - %s\r\n", viper.Get("default_pce_name").(string), viper.Get(viper.Get("default_pce_name").(string)+".fqdn").(string))

		utils.Log(0, "get-default command completed")

	},
}

// PCEListCmd gets all PCEs
var PCEListCmd = &cobra.Command{
	Use:   "pce-list",
	Short: "List all PCEs in pce.yaml.",
	PreRun: func(cmd *cobra.Command, args []string) {
		configFilePath, err = filepath.Abs(viper.ConfigFileUsed())
		if err != nil {
			utils.Log(1, err.Error())
		}
	},
	Run: func(cmd *cobra.Command, args []string) {

		utils.Log(0, "pce-list command started")

		allSettings := viper.AllSettings()

		defaultPCEName := viper.Get("default_pce_name").(string)

		for k := range allSettings {
			if viper.Get(k+".fqdn") != nil {
				if k == defaultPCEName {
					fmt.Printf("%s (%s) - DEFAULT\r\n", k, viper.Get(k+".fqdn").(string))
				} else {
					fmt.Printf("%s (%s)\r\n", k, viper.Get(k+".fqdn").(string))
				}
			}
		}

		utils.Log(0, "pce-list command completed")

	},
}
