package wkldexport

import (
	"fmt"
	"math"
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
var managedOnly, unmanagedOnly, includeVuln, removeDescNewLines bool
var outputFileName string

func init() {
	WkldExportCmd.Flags().BoolVarP(&managedOnly, "managed-only", "m", false, "only export managed workloads.")
	WkldExportCmd.Flags().BoolVarP(&unmanagedOnly, "unmanaged-only", "u", false, "only export unmanaged workloads.")
	WkldExportCmd.Flags().BoolVarP(&includeVuln, "incude-vuln-data", "v", false, "include vulnerability data.")
	WkldExportCmd.Flags().StringVar(&outputFileName, "output-file", "", "optionally specify the name of the output file location. default is current location with a timestamped filename.")
	WkldExportCmd.Flags().BoolVar(&removeDescNewLines, "remove-desc-newline", false, "will remove new line characters in description field.")

}

// WkldExportCmd runs the workload identifier
var WkldExportCmd = &cobra.Command{
	Use:   "wkld-export",
	Short: "Create a CSV export of all workloads in the PCE.",
	Long: `
Create a CSV export of all workloads in the PCE.

The update-pce and --no-prompt flags are ignored for this command.`,
	Run: func(cmd *cobra.Command, args []string) {

		// Get the PCE
		pce, err = utils.GetTargetPCE(true)
		if err != nil {
			utils.LogError(err.Error())
		}

		exportWorkloads()
	},
}

func exportWorkloads() {

	// Log command execution
	utils.LogStartCommand("wkld-export")

	// Start the data slice with headers
	headers := []string{HeaderHostname, HeaderName, HeaderManaged, HeaderRole, HeaderApp, HeaderEnv, HeaderLoc, HeaderInterfaces, HeaderPublicIP, HeaderMachineAuthenticationID, HeaderIPWithDefaultGw, HeaderNetmaskOfIPWithDefGw, HeaderDefaultGw, HeaderDefaultGwNetwork, HeaderSPN, HeaderHref, HeaderDescription, HeaderPolicyState, HeaderVisibilityState, HeaderOnline, HeaderAgentStatus, HeaderAgentHealth, HeaderSecurityPolicySyncState, HeaderSecurityPolicyAppliedAt, HeaderSecurityPolicyReceivedAt, HeaderSecurityPolicyRefreshAt, HeaderLastHeartbeatOn, HeaderHoursSinceLastHeartbeat, HeaderCreatedAt, HeaderOsID, HeaderOsDetail, HeaderVenHref, HeaderAgentVersion, HeaderAgentID, HeaderActivePceFqdn, HeaderServiceProvider, HeaderDataCenter, HeaderDataCenterZone, HeaderCloudInstanceID}
	if includeVuln {
		headers = append(headers, HeaderVulnExposureScore, HeaderNumVulns, HeaderMaxVulnScore, HeaderVulnScore, HeaderVulnPortExposure, HeaderAnyVulnExposure, HeaderIpListVulnExposure)
	}
	csvData := [][]string{append(headers, HeaderExternalDataSet, HeaderExternalDataReference)}
	stdOutData := [][]string{{HeaderHostname, HeaderRole, HeaderApp, HeaderEnv, HeaderLoc, HeaderPolicyState}}

	// GetAllWorkloads
	qp := make(map[string]string)
	if unmanagedOnly {
		qp["managed"] = "false"
	}
	if managedOnly {
		qp["managed"] = "true"
	}
	qp["representation"] = "workload_labels_vulnerabilities"
	wklds, a, err := pce.GetAllWorkloadsQP(qp)
	utils.LogAPIResp("GetAllWorkloads", a)
	if err != nil {
		utils.LogError(fmt.Sprintf("getting all workloads - %s", err))
	}

	for _, w := range wklds {

		// Skip deleted workloads
		if *w.Deleted {
			continue
		}

		// Get interfaces
		interfaces := []string{}
		for _, i := range w.Interfaces {
			ipAddress := fmt.Sprintf("%s:%s", i.Name, i.Address)
			if i.CidrBlock != nil && *i.CidrBlock != 0 {
				ipAddress = fmt.Sprintf("%s:%s/%s", i.Name, i.Address, strconv.Itoa(*i.CidrBlock))
			}
			interfaces = append(interfaces, ipAddress)
		}

		// Get Managed Status
		managedStatus := false
		if (w.Agent != nil && w.Agent.Href != "") || (w.VEN != nil && w.VEN.Href != "") {
			managedStatus = true
		}

		// Assume the VEN-dependent fields are unmanaged
		policySyncStatus := "unmanaged"
		policyAppliedAt := "unmanaged"
		poicyReceivedAt := "unmanaged"
		policyRefreshAt := "unmanaged"
		venVersion := "unmanaged"
		lastHeartBeat := "unmanaged"
		hoursSinceLastHB := "unmanaged"
		venID := "unmanaged"
		pairedPCE := "unmanaged"
		agentStatus := "unmanaged"
		instanceID := "unmanaged"
		agentHealth := "unmanaged"
		venHref := "unmanaged"
		// If it is managed, get that information
		if w.Agent != nil && w.Agent.Href != "" {
			policySyncStatus = w.Agent.Status.SecurityPolicySyncState
			policyAppliedAt = w.Agent.Status.SecurityPolicyAppliedAt
			poicyReceivedAt = w.Agent.Status.SecurityPolicyReceivedAt
			policyRefreshAt = w.Agent.Status.SecurityPolicyRefreshAt
			venVersion = w.Agent.Status.AgentVersion
			lastHeartBeat = w.Agent.Status.LastHeartbeatOn
			hoursSinceLastHB = fmt.Sprintf("%f", w.HoursSinceLastHeartBeat())
			venID = w.Agent.GetID()
			pairedPCE = w.Agent.ActivePceFqdn
			if pairedPCE == "" {
				pairedPCE = pce.FQDN
			}
			agentStatus = w.Agent.Status.Status
			instanceID = w.Agent.Status.InstanceID
			if instanceID == "" {
				instanceID = "NA"
			}
			if w.Agent.Status.AgentHealth != nil && len(w.Agent.Status.AgentHealth) > 0 {
				healthSlice := []string{}
				for _, a := range w.Agent.Status.AgentHealth {
					healthSlice = append(healthSlice, fmt.Sprintf("%s (%s)", a.Type, a.Severity))
				}
				agentHealth = strings.Join(healthSlice, "; ")
			} else {
				agentHealth = "NA"
			}
		}

		// Start using VEN properties
		if w.VEN != nil {
			venHref = w.VEN.Href
		}

		// Remove newlines in description
		if removeDescNewLines {
			w.Description = utils.ReplaceNewLine(w.Description)
		}

		// Append to data slice
		data := []string{w.Hostname, w.Name, strconv.FormatBool(managedStatus), w.GetRole(pce.Labels).Value, w.GetApp(pce.Labels).Value, w.GetEnv(pce.Labels).Value, w.GetLoc(pce.Labels).Value, strings.Join(interfaces, ";"), w.PublicIP, w.DistinguishedName, w.GetIPWithDefaultGW(), w.GetNetMaskWithDefaultGW(), w.GetDefaultGW(), w.GetNetworkWithDefaultGateway(), w.ServicePrincipalName, w.Href, w.Description, w.GetMode(), w.GetVisibilityLevel(), strconv.FormatBool(w.Online), agentStatus, agentHealth, policySyncStatus, policyAppliedAt, poicyReceivedAt, policyRefreshAt, lastHeartBeat, hoursSinceLastHB, w.CreatedAt, w.OsID, w.OsDetail, venHref, venVersion, venID, pairedPCE, w.ServiceProvider, w.DataCenter, w.DataCenterZone, instanceID}

		if includeVuln {
			var numVulns, maxVulnScore, vulnScore, vulnPortExposure, anyExposure, iplExposure, vulnExposureScore string
			targets := []*string{&maxVulnScore, &vulnScore, &vulnExposureScore}
			if w.VulnerabilitySummary != nil {
				values := []int{w.VulnerabilitySummary.MaxVulnerabilityScore, w.VulnerabilitySummary.VulnerabilityScore, w.VulnerabilitySummary.VulnerabilityExposureScore}
				for i, t := range targets {
					*t = strconv.Itoa(int((math.Round((float64(values[i]) / float64(10))))))
				}

				numVulns = strconv.Itoa(w.VulnerabilitySummary.NumVulnerabilities)
				vulnPortExposure = strconv.Itoa(w.VulnerabilitySummary.VulnerablePortExposure)
				anyExposure = strconv.FormatBool(w.VulnerabilitySummary.VulnerablePortWideExposure.Any)
				iplExposure = strconv.FormatBool(w.VulnerabilitySummary.VulnerablePortWideExposure.IPList)
			}
			data = append(data, vulnExposureScore, numVulns, maxVulnScore, vulnScore, vulnPortExposure, anyExposure, iplExposure, w.ExternalDataSet, w.ExternalDataReference)
		}
		data = append(data, w.ExternalDataSet, w.ExternalDataReference)
		csvData = append(csvData, data)

		stdOutData = append(stdOutData, []string{w.Hostname, w.GetRole(pce.Labels).Value, w.GetApp(pce.Labels).Value, w.GetEnv(pce.Labels).Value, w.GetLoc(pce.Labels).Value, w.GetMode()})
	}

	if len(csvData) > 1 {
		if outputFileName == "" {
			outputFileName = fmt.Sprintf("workloader-wkld-export-%s.csv", time.Now().Format("20060102_150405"))
		}
		utils.WriteOutput(csvData, stdOutData, outputFileName)
		utils.LogInfo(fmt.Sprintf("%d workloads exported", len(csvData)-1), true)
	} else {
		// Log command execution for 0 results
		utils.LogInfo("no workloads in PCE.", true)
	}

	utils.LogEndCommand("wkld-export")

}
