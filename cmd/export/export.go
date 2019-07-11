package export

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/brian1917/illumioapi"

	"github.com/brian1917/workloader/utils"
	"github.com/spf13/cobra"
)

// Declare local global variables
var pce illumioapi.PCE
var err error

// ExportCmd runs the workload identifier
var ExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Create a CSV export of all workloads in the PCE.",
	Run: func(cmd *cobra.Command, args []string) {

		pce, err = utils.GetPCE()
		if err != nil {
			utils.Log(1, fmt.Sprintf("getting PCE for export command - %s", err))
		}

		exportWorkloads()
	},
}

func exportWorkloads() {

	// Start the data slice with headers
	data := [][]string{[]string{"hostname", "name", "href", "online", "os_id", "role", "app", "env", "loc", "status", "interfaces", "public_ip", "ven_version"}}

	// GetAllWorkloads
	wklds, _, err := illumioapi.GetAllWorkloads(pce)
	if err != nil {
		utils.Log(1, fmt.Sprintf("getting all workloads - %s", err))
	}

	// Get LabelMap
	labelMap, err := illumioapi.GetLabelMapH(pce)
	if err != nil {
		utils.Log(1, fmt.Sprintf("getting label map - %s", err))
	}

	for _, w := range wklds {

		// Skip deleted workloads
		if *w.Deleted {
			continue
		}

		// Set up variables
		interfaces := []string{}
		status := ""
		venVersion := ""

		// Get interfaces
		for _, i := range w.Interfaces {
			interfaces = append(interfaces, i.Name+":"+i.Address+"/"+strconv.Itoa(i.CidrBlock))
		}

		// Set status and VEN version
		if w.Agent == nil || w.Agent.Href == "" {
			status = "Unmanaged"
			venVersion = "Unmanaged"
		} else {
			status = w.Agent.Config.Mode
			venVersion = w.Agent.Status.AgentVersion
			if w.Agent.Config.Mode == "illuminated" && !w.Agent.Config.LogTraffic {
				status = "Build"
			}

			if w.Agent.Config.Mode == "illuminated" && w.Agent.Config.LogTraffic {
				status = "Test"
			}
		}

		// Append to data slice
		data = append(data, []string{w.Hostname, w.Name, w.Href, strconv.FormatBool(w.Online), w.OsID, w.GetRole(labelMap).Value, w.GetApp(labelMap).Value, w.GetEnv(labelMap).Value, w.GetLoc(labelMap).Value, status, strings.Join(interfaces, ";"), w.PublicIP, venVersion})
	}

	if len(data) > 1 {
		// Create output file
		timestamp := time.Now().Format("20060102_150405")
		outFile, err := os.Create("workloads-export-" + timestamp + ".csv")

		if err != nil {
			utils.Logger.Fatalf("[ERROR] - Creating file - %s\n", err)
		}

		// Write CSV data
		writer := csv.NewWriter(outFile)
		writer.WriteAll(data)
		if err := writer.Error(); err != nil {
			utils.Logger.Fatalf("[ERROR] - writing csv - %s", err)
		}
	}

	fmt.Printf("Exported %d workloads.\r\n", len(data)-1)
}
