package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"
)

var (
	upgradeProfile = flag.String("upgrade-profile", "", "UpgradeProfile of aks. Will be always sorted by aks version.")
)

//AksProfiles for available aks version to install and upgrade
type AksProfiles []struct {
	Upgrades []Upgrades `json:"upgrades"`
	Version  string     `json:"version"`
}

//Upgrades profiles for given aks version
type Upgrades struct {
	IsPreview           bool   `json:"isPreview"`
	Orchestratortype    string `json:"orchestratorType"`
	Orchestratorversion string `json:"orchestratorVersion"`
}

func main() {
	flag.Parse()

	aksVersions := []string{}
	currentVersion := ""

	aksProfiles := AksProfiles{}
	json.Unmarshal([]byte(*upgradeProfile), &aksProfiles)

	if len(aksProfiles) > 0 {
		aksVersions = append(aksVersions, aksProfiles[0].Version) //Since input is always sorted, first version is always highest.
		currentVersion = aksProfiles[0].Version
	}

	for _, profile := range aksProfiles {
		if len(aksVersions) == 3 { //Capture 3 versions required for n, n+1, n+2 upgrade test
			break
		}

		if !contains(aksVersions, strings.Join(strings.Split(profile.Version, ".")[:2], ".")) { //Check if we have already captured major.minor version
			for _, upgrade := range profile.Upgrades {
				if upgrade.Orchestratorversion == currentVersion && !upgrade.IsPreview { //Check if previously captured version can be upgraded from this version
					aksVersions = append(aksVersions, profile.Version)
					currentVersion = profile.Version
					break
				}
			}
		}
	}

	fmt.Printf("%v\n", strings.Join(aksVersions, "~"))
}

func contains(versions []string, versionPrefix string) bool {
	for _, version := range versions {
		if strings.HasPrefix(version, versionPrefix) {
			return true
		}
	}
	return false
}
