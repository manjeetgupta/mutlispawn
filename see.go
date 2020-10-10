package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"time"
)

var redundantDeploymentMapFilePath string = "/root/Sync/redundantDeploymentMap.json"
var redundantDeploymentMap = map[string]uint16{}

//Tag:2
func readRedundantDeploymentMapFile() {

	jsonFile, err := os.Open(redundantDeploymentMapFilePath)
	defer jsonFile.Close()

	if err != nil {
		fmt.Println("File open error:", err)
	} else {
		jsonString, _ := ioutil.ReadAll(jsonFile)
		err = json.Unmarshal(jsonString, &redundantDeploymentMap)
		if err != nil {
			fmt.Println("Unmarshalling error:", err)
			return
		}

		appNames := make([]string, len(redundantDeploymentMap))
		i := 0
		for key := range redundantDeploymentMap {
			appNames[i] = key
			i++
		}
		sort.Strings(appNames)

		fmt.Println("    7654321076543210")
		for _, key := range appNames {
			fmt.Printf("%s :%016b\n", key, redundantDeploymentMap[key])
		}

		// for key, v := range redundantDeploymentMap {
		// 	fmt.Printf("%s :%016b\n", key, v)
		// }
	}
}

func main() {

	for {
		fmt.Println("\n...",time.Now().Format(time.Stamp),"..................................")
		readRedundantDeploymentMapFile()
		time.Sleep(1000 * time.Millisecond)
	}
}
