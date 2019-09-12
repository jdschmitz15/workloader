package login

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/brian1917/workloader/utils"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Set global variables for flags
var clear bool

func init() {
	LogoutCmd.Flags().BoolVarP(&clear, "clear-keys", "x", false, "Remove existing JSON authentication file and clear all Workloader generated API credentials from the PCE.")
}

// LogoutCmd removes the pce.yaml file
var LogoutCmd = &cobra.Command{
	Use:  "logout",
	Long: fmt.Sprintf("\r\n%s\r\n\r\nThe --update-pce and --no-prompt flags are ignored for this command.", utils.LogOutDesc()),
	PreRun: func(cmd *cobra.Command, args []string) {
		configFilePath, err = filepath.Abs(viper.ConfigFileUsed())
		if err != nil {
			utils.Log(1, err.Error())
		}
	},
	Run: func(cmd *cobra.Command, args []string) {

		// Get the debug value from viper
		debug = viper.Get("debug").(bool)

		logout()
	},
}

func logout() {

	utils.Log(0, "logout command started")

	// Start by clearing API keys
	if clear {

		// Log start of command
		utils.Log(0, "removing API keys...")

		// Get the PCE
		pce, err := utils.GetPCE()
		if err != nil {
			utils.Log(1, err.Error())
		}

		// Get all API Keys
		apiKeys, _, err := pce.GetAllAPIKeys(viper.Get("userhref").(string))
		if err != nil {
			utils.Log(1, err.Error())
		}

		// Delete the API keys that are from Workloader
		for _, a := range apiKeys {
			if a.Name == "Workloader" {
				_, err := pce.DeleteHref(a.Href)
				if err != nil {
					utils.Log(1, err.Error())
				}
				utils.Log(0, fmt.Sprintf("deleted %s", a.Href))
			}
		}
	}

	// Remove the YAML file
	utils.Log(0, fmt.Sprintf("location of authentication file is %s", configFilePath))
	if runtime.GOOS == "windows" {
		viper.Set("key", "")
		viper.Set("user", "")
		viper.WriteConfig()
		fmt.Printf("Removed login info from %s\r\n", configFilePath)
		utils.Log(0, fmt.Sprintf("removed login info from %s", configFilePath))
	} else {
		if err := os.Remove(configFilePath); err != nil {
			utils.Log(1, fmt.Sprintf("error deleting file - %s", err))
		}
		utils.Log(0, fmt.Sprintf("successfully deleted %s", configFilePath))
	}

}