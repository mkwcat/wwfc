package serverbrowser

import (
	"github.com/logrusorgru/aurora/v3"
	"regexp"
	"strconv"
	"wwfc/logging"
)

// TODO: Even if we don't use it in the end, we could still implement parsing the filter string.
// DWC makes requests in the following two formats:
// Matching: dwc_mver = %d and dwc_pid != %u and maxplayers = %d and numplayers < %d and dwc_mtype = %d and dwc_hoststate = %u and dwc_suspend = %u and (%s)
// ...OR
// Self Lookup: dwc_pid = %u

// Example: dwc_mver = 90 and dwc_pid != 43 and maxplayers = 11 and numplayers < 11 and dwc_mtype = 0 and dwc_hoststate = 0 and dwc_suspend = 0 and (rk = 'vs' and ev >= 4250 and ev <= 5750 and p = 0)

var (
	regexSelfLookup  = regexp.MustCompile(`^dwc_pid = (\d{1,10})$`)
	regexMatchmaking = regexp.MustCompile(`^dwc_mver = -?(\d{1,10}) and dwc_pid != (\d{1,10}) and maxplayers = -?(\d{1,10}) and numplayers < -?(\d{1,10}) and dwc_mtype = -?(\d{1,10}) and dwc_hoststate = (\d{1,10}) and dwc_suspend = (\d{1,10}) and \((.*)\)$`)
)

func filterServers(servers []map[string]string, queryGame string, filter string, publicIP string) []map[string]string {
	if match := regexSelfLookup.FindStringSubmatch(filter); match != nil {
		dwc_pid := match[1]

		filtered := []map[string]string{}

		// Search for where the profile ID matches
		for _, server := range servers {
			if server["dwc_pid"] == dwc_pid {
				logging.Info(ModuleName, "Self lookup from", aurora.Cyan(dwc_pid), "ok")
				return []map[string]string{server}
			}

			// Alternatively, if the server hasn't set its dwc_pid field yet, we return servers matching the request's public IP.
			// If multiple servers exist with the same public IP then the client will use the one with the matching port.
			// This is a bit of a hack to speed up server creation.
			if _, ok := server["dwc_pid"]; !ok && server["publicip"] == publicIP {
				// Create a copy of the map with some values changed
				newServer := map[string]string{}
				for k, v := range server {
					newServer[k] = v
				}
				newServer["dwc_pid"] = dwc_pid
				newServer["dwc_mtype"] = "0"
				newServer["dwc_mver"] = "0"
				filtered = append(filtered, newServer)
			}
		}

		if len(filtered) == 0 {
			logging.Error(ModuleName, "Could not find server with dwc_pid", aurora.Cyan(dwc_pid))
			return []map[string]string{}
		}

		logging.Info(ModuleName, "Self lookup for", aurora.Cyan(dwc_pid), "matched", aurora.BrightCyan(len(filtered)), "servers via public IP")
		return filtered
	}

	if match := regexMatchmaking.FindStringSubmatch(filter); match != nil {
		dwc_mver := match[1]
		dwc_pid := match[2]
		maxplayers := match[3]
		numplayers, err := strconv.ParseInt(match[4], 10, 32)
		if err != nil {
			logging.Error(ModuleName, "Invalid numplayers:", aurora.Cyan(match[4]), "from", aurora.Cyan(match))
			return []map[string]string{}
		}
		dwc_mtype := match[5]
		dwc_hoststate := match[6]
		dwc_suspend := match[7]
		// gameFilter := match[8]

		filtered := []map[string]string{}

		// Find servers that match the requested parameters
		// TODO: Handle game specific filters (i.e. Regionals or MKW VR search)
		for _, server := range servers {
			if server["dwc_mver"] == dwc_mver && server["dwc_pid"] != dwc_pid && server["maxplayers"] == maxplayers && server["dwc_mtype"] == dwc_mtype && server["dwc_hoststate"] == dwc_hoststate && server["dwc_suspend"] == dwc_suspend {
				server_numplayers, err := strconv.ParseInt(server["numplayers"], 10, 32)
				if err != nil {
					logging.Error(ModuleName, "Invalid numplayers:", aurora.Cyan(match[4]))
					continue
				}

				if server_numplayers < numplayers {
					filtered = append(filtered, server)
				}
			}
		}

		logging.Info(ModuleName, "Matched", aurora.BrightCyan(len(filtered)), "servers")
		return filtered
	}

	logging.Error(ModuleName, "Unable to match filter for", aurora.Cyan(filter))
	return []map[string]string{}
}