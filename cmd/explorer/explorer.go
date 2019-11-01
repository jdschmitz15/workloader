package explorer

import (
	"fmt"
	"strconv"
	"time"

	"github.com/brian1917/illumioapi"
	"github.com/brian1917/workloader/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var app, exclServiceObj, exclServiceCSV, start, end string
var exclAllowed, exclPotentiallyBlocked, exclBlocked, appGroupLoc, ignoreIPGroup, consolidate, debug bool
var threshold int
var pce illumioapi.PCE
var err error

func init() {

	ExplorerCmd.Flags().StringVarP(&app, "limit-to-app", "a", "", "app name to limit Explorer results to flows with that app as a provider or a consumer. default is all apps.")
	// NetTrafficCmd.Flags().StringVar(&exclServiceObj, "exclude-service-object", "", "service name to exclude in explorer query (only port/proto and port ranges are excluded).")
	ExplorerCmd.Flags().StringVarP(&exclServiceCSV, "exclude-service-csv", "x", "", "file location of csv with port/protocols to exclude. CSV should have NO HEADERS with port number in column 1 and IANA numeric protocol in Col 2.")
	ExplorerCmd.Flags().StringVarP(&start, "start", "s", time.Date(time.Now().Year()-5, time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC).Format("2006-01-02"), "start date in the format of yyyy-mm-dd. Date is set as midnight UTC.")
	ExplorerCmd.Flags().StringVarP(&end, "end", "e", time.Now().Add(time.Hour*24).Format("2006-01-02"), "end date in the format of yyyy-mm-dd. Date is set as midnight UTC.")
	ExplorerCmd.Flags().BoolVar(&exclAllowed, "excl-allowed", false, "excludes allowed traffic flows.")
	ExplorerCmd.Flags().BoolVar(&exclPotentiallyBlocked, "excl-potentially-blocked", false, "excludes potentially blocked traffic flows.")
	ExplorerCmd.Flags().BoolVar(&exclBlocked, "excl-blocked", false, "excludes blocked traffic flows.")
	ExplorerCmd.Flags().IntVar(&threshold, "threshold", 90000, "threshold to start iterating.")
	ExplorerCmd.Flag("threshold").Hidden = true
	ExplorerCmd.Flags().SortFlags = false
}

// ExplorerCmd summarizes flows
var ExplorerCmd = &cobra.Command{
	Use:   "explorer",
	Short: "Export explorer traffic data and include workload subnet and default gateway.",
	Long: `
Export explorer traffic data and include workload subnet and default gateway.

To filter unwanted traffic, create a CSV with NO HEADERS. Column 1 should have port number and column 2 should have the IANA protocol number and pass the csv file into the --exclude-service-csv (-x) flag.`,
	Run: func(cmd *cobra.Command, args []string) {

		pce, err = utils.GetPCE(true)
		if err != nil {
			utils.Log(1, err.Error())
		}

		// Get the debug value from viper
		debug = viper.Get("debug").(bool)

		explorerExport()
	},
}

func explorerExport() {

	// Set threshold
	illumioapi.Threshold = threshold

	// Log start
	utils.Log(0, "started explorer command")

	// Build policy status slice
	var pStatus []string
	if !exclAllowed {
		pStatus = append(pStatus, "allowed")
	}
	if !exclPotentiallyBlocked {
		pStatus = append(pStatus, "potentially_blocked")
	}
	if !exclBlocked {
		pStatus = append(pStatus, "blocked")
	}
	utils.Log(0, fmt.Sprintf("pStatus = %#v", pStatus))

	// Get the state and end date
	startDate, err := time.Parse(fmt.Sprintf("2006-01-02 MST"), fmt.Sprintf("%s %s", start, "UTC"))
	if err != nil {
		utils.Log(1, err.Error())
	}
	startDate = startDate.In(time.UTC)
	utils.Log(0, fmt.Sprintf("startDate = %v", startDate))

	endDate, err := time.Parse(fmt.Sprintf("2006-01-02 MST"), fmt.Sprintf("%s %s", end, "UTC"))
	if err != nil {
		utils.Log(1, err.Error())
	}
	endDate = endDate.In(time.UTC)
	utils.Log(0, fmt.Sprintf("endDate = %v", endDate))

	// Create the default query struct
	tq := illumioapi.TrafficQuery{
		StartTime:      startDate,
		EndTime:        endDate,
		PolicyStatuses: pStatus,
		MaxFLows:       100000}

	// If exclude service is provided, add it to the traffic query
	if exclServiceCSV != "" {
		tq.PortProtoExclude = utils.GetServicePortsCSV(exclServiceCSV)
	}

	// If an app is provided, adjust query to include it
	if app != "" {
		label, a, err := pce.GetLabelbyKeyValue("app", app)
		if debug {
			utils.LogAPIResp("GetLabelbyKeyValue", a)
		}
		if err != nil {
			utils.Log(1, fmt.Sprintf("getting label HREF - %s", err))
		}
		if label.Href == "" {
			utils.Log(1, fmt.Sprintf("%s does not exist as an app label.", app))
		}
		tq.SourcesInclude = []string{label.Href}
	}

	utils.Log(0, fmt.Sprintf("traffic query object: %+v", tq))

	// Run traffic query
	traffic, err := pce.IterateTraffic(tq, true)
	if err != nil {
		utils.Log(1, err.Error())
	}

	// If app is provided, switch to the destination include, clear the sources include, run query again, append to previous result
	if app != "" {
		tq.DestinationsInclude = tq.SourcesInclude
		tq.SourcesInclude = []string{}
		utils.Log(0, fmt.Sprintf("second traffic query object: %+v", tq))
		traffic2, err := pce.IterateTraffic(tq, true)
		if err != nil {
			utils.Log(1, fmt.Sprintf("making second explorer API call - %s", err))
		}
		traffic = append(traffic, traffic2...)
	}

	// Build our CSV structure
	data := [][]string{[]string{"src_ip", "src_net_mask", "src_default_gw", "src_hostname", "src_role", "src_app", "src_env", "src_loc", "dst_ip", "dst_net_mask", "dst_default_gw", "dst_hostname", "dst_role", "dst_app", "dst_env", "dst_loc", "port", "protocol", "policy_status", "date_first", "date_last", "num_flows"}}

	// Get LabelMap for getting workload labels
	_, err = pce.GetLabelMaps()
	if err != nil {
		utils.Log(1, err.Error())
	}

	// Get WorkloadMap by hostname
	whm, _, err := pce.GetWkldHostMap()
	if err != nil {
		utils.Log(1, err.Error())
	}

	// Add each traffic entry to the data slic
	for _, t := range traffic {
		// Source
		src := []string{t.Src.IP, "NA", "NA", "NA", "NA", "NA", "NA", "NA"}
		if t.Src.Workload != nil {
			src = []string{t.Src.IP, wkldNetMask(t.Src.Workload.Hostname, t.Src.IP, whm), wkldGW(t.Src.Workload.Hostname, whm), t.Src.Workload.Hostname, t.Src.Workload.GetRole(pce.LabelMapH).Value, t.Src.Workload.GetApp(pce.LabelMapH).Value, t.Src.Workload.GetEnv(pce.LabelMapH).Value, t.Src.Workload.GetLoc(pce.LabelMapH).Value}
		}

		// Destination
		dst := []string{t.Dst.IP, "NA", "NA", "NA", "NA", "NA", "NA", "NA"}
		if t.Dst.Workload != nil {
			dst = []string{t.Dst.IP, wkldNetMask(t.Dst.Workload.Hostname, t.Dst.IP, whm), wkldGW(t.Dst.Workload.Hostname, whm), t.Dst.Workload.Hostname, t.Dst.Workload.GetRole(pce.LabelMapH).Value, t.Dst.Workload.GetApp(pce.LabelMapH).Value, t.Dst.Workload.GetEnv(pce.LabelMapH).Value, t.Dst.Workload.GetLoc(pce.LabelMapH).Value}
		}

		// Append source, destination, port, protocol, policy decision, time stamps, and number of connections to data
		protocols := illumioapi.ProtocolList()
		d := append(src, dst...)
		d = append(d, strconv.Itoa(t.ExpSrv.Port))
		d = append(d, protocols[t.ExpSrv.Proto])
		d = append(d, t.PolicyDecision)
		d = append(d, t.TimestampRange.FirstDetected)
		d = append(d, t.TimestampRange.LastDetected)
		d = append(d, strconv.Itoa(t.NumConnections))
		data = append(data, d)
	}

	// Write the data
	utils.WriteOutput(data, data, fmt.Sprintf(fmt.Sprintf("workloader-explorer-%s.csv", time.Now().Format("20060102_150405"))))

	// Log end
	utils.Log(0, "explorer command complete")

}

func wkldGW(hostname string, wkldHostMap map[string]illumioapi.Workload) string {
	if wkld, ok := wkldHostMap[hostname]; ok {
		return wkld.GetDefaultGW()
	}
	return "NA"
}

func wkldNetMask(hostname, ip string, wkldHostMap map[string]illumioapi.Workload) string {
	if wkld, ok := wkldHostMap[hostname]; ok {
		return wkld.GetNetMask(ip)
	}
	return "NA"
}