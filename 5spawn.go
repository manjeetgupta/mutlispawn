package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var applicaitonpidStateMap = map[string]pidState{}
var runtimedeploymentMap = map[int]appDetailruntime{}
var deploymentConfigurationMap map[string][]appDetail
var ch = make(chan int, 3)
var wg sync.WaitGroup
var stateFileName string

type appDetailruntime struct {
	CsciName        string
	ListOfArguments string
}

type pidState struct {
	Pid   int `json:"pid"`
	State int `json:"state"`
	//Struct fields must start with upper case letter (exported) for the JSON package to see their value.
}

//===============Deployment map related structures============
type appList struct {
	XMLName xml.Name    `xml:"appList"`
	App     []appDetail `xml:"appDetail"`
}

type appDetail struct {
	XMLName         xml.Name `xml:"appDetail"`
	CsciName        string   `xml:"csci_name"`
	AirbaseID       int      `xml:"airbase_id"`
	HardwareID      string   `xml:"hardware_id"`
	HardwareType    string   `xml:"hardware_type"`
	CsciID          int      `xml:"csci_id"`
	MaxRetries      int      `xml:"max_retries"`
	Redundant       bool     `xml:"redundant"`
	ListOfArguments string   `xml:"list_of_arguments"`
}

func readXML(filename string) appList {

	fmt.Println("<>Inside readXML funtion")
	xmlFile, err := os.Open(filename)
	if err != nil {
		fmt.Println("5*---", err)
	}

	fmt.Println("5----Successfully opened", filename)
	defer xmlFile.Close()

	byteValue, _ := ioutil.ReadAll(xmlFile)

	var deploymentFileContent appList
	xml.Unmarshal(byteValue, &deploymentFileContent)

	/*
		for i := 0; i < len(deploymentFileContent.App); i++ {
			fmt.Println("Entry: ", i)
			fmt.Println("Csci_name: ", deploymentFileContent.App[i].CsciName)
			fmt.Println("Airbase_id: ", deploymentFileContent.App[i].AirbaseID)
			fmt.Println("Hardware_id: ", deploymentFileContent.App[i].HardwareID)
			fmt.Println("Hardware_type: ", deploymentFileContent.App[i].HardwareType)
			fmt.Println("Csci_id: ", deploymentFileContent.App[i].CsciID)
			fmt.Println("Max_retries: ", deploymentFileContent.App[i].MaxRetries)
			fmt.Println("Redundant: ", deploymentFileContent.App[i].Redundant)
			fmt.Println("List_of_arguments: ", deploymentFileContent.App[i].ListOfArguments)
			fmt.Println("--------------------------------------------")
		}
	*/
	fmt.Println("<>Leaving readXML funtion")
	return deploymentFileContent
}

//===========================

func isPid(pid int) bool {

	fmt.Println("<>Inside isPid funtion")

	if pid <= 0 {
		fmt.Println("4*---Invalid pid", pid)
		return false
	}
	proc, err := os.FindProcess(int(pid))
	if err != nil {
		fmt.Println("4*---", err)
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}

	if err.Error() == "os: process already finished" {
		return false
	}
	errno, ok := err.(syscall.Errno)
	if !ok {
		fmt.Println("4*---", err)
		return false
	}

	switch errno {
	case syscall.ESRCH:
		return false
	case syscall.EPERM:
		return true
	}

	fmt.Println("<>Leaving isPid funtion")
	return false
}

func storeMap(applicaitonpidStateMap map[string]pidState) {

	fmt.Println("<>Inside storeMap funtion")
	f, err := os.Create(stateFileName)
	if err != nil {
		fmt.Println("3*---", err)
		return
	}

	jsonString, err := json.Marshal(applicaitonpidStateMap)
	if err != nil {
		fmt.Println("3*---", err)
		return
	}
	fmt.Println("3----Marshalled Map:", string(jsonString))

	l, err := f.WriteString(string(jsonString))
	if err != nil {
		fmt.Println("3*---", err)
		f.Close()
		return
	}
	fmt.Printf("3----%v Bytes successfully in %v\n", l, stateFileName)

	err = f.Close()
	if err != nil {
		fmt.Println("3*---", err)
		return
	}
	fmt.Println("<>Leaving storeMap funtion")
}

func spawnApp(name string, arg string) (chan int, error) {

	fmt.Println("<>Inside spawnApp funtion")
	cmd := exec.Command(name, arg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout

	if err := cmd.Start(); err != nil {
		fmt.Println("2----Error in start()")
		return nil, err
	}

	runtimedeploymentMap[cmd.Process.Pid] = appDetailruntime{CsciName: name, ListOfArguments: arg}
	applicaitonpidStateMap[name] = pidState{Pid: cmd.Process.Pid, State: 0}
	fmt.Println("2----Updated maps after spawning are below: ")
	fmt.Println("2----", applicaitonpidStateMap)
	fmt.Println("2----", runtimedeploymentMap)
	storeMap(applicaitonpidStateMap)

	wg.Add(1)
	go func() {
		fmt.Println("2----Waiting for", cmd.Process.Pid)
		cmd.Wait()
		defer wg.Done()
		if _, found := runtimedeploymentMap[cmd.Process.Pid]; found { //if not checked then both will waits ( line 322 and this one)
			fmt.Println("2----Pid present in runtimedeploymentMap,so writing to channel")
			ch <- cmd.Process.Pid
		}
	}()

	fmt.Println("<>Leaving spawnApp funtion")
	return ch, nil

}

func main() {

	fmt.Println("<>Inside main funtion")
	var pidExistStatus bool = false
	arg := os.Args[1]

	//========================Creating Deployment map=========================
	var deploymentFileContent appList
	deploymentFileContent = readXML("DeploymentConfiguration.xml")
	deploymentConfigurationMap = make(map[string][]appDetail)

	for i := 0; i < len(deploymentFileContent.App); i++ {
		s := strings.TrimSpace(deploymentFileContent.App[i].HardwareID)
		listOfHardware := strings.Split(s, ",")
		for j := range listOfHardware {
			hw := listOfHardware[j]
			combinedKey := strconv.Itoa(deploymentFileContent.App[i].AirbaseID) + hw
			value, ok := deploymentConfigurationMap[combinedKey]
			if ok == true {
				//fmt.Println("1----",combinedKey,":key already present in map")
				value = append(value, deploymentFileContent.App[i])
				deploymentConfigurationMap[combinedKey] = value
			} else {
				//fmt.Println("1----",combinedKey,":new key")
				var value []appDetail
				value = append(value, deploymentFileContent.App[i])
				deploymentConfigurationMap[combinedKey] = value
			}
		}
	}
	//fmt.Println("1----Complete DeploymentMap:\n", deploymentConfigurationMap)
	fmt.Println("1----Hardware", arg, "is configured for following applications:")
	element := deploymentConfigurationMap["1"+arg]
	for _, apptospawn := range element {
		//fmt.Println("1----", apptospawn)
		fmt.Println("1----", apptospawn.CsciName)
	}

	//=================================================

	stateFileName = "ClientState" + arg + ".json"
	jsonFile, err := os.Open(stateFileName)
	defer jsonFile.Close()
	if err != nil {
		fmt.Println("1*---", err)
	} else {
		jsonString, _ := ioutil.ReadAll(jsonFile)
		err = json.Unmarshal(jsonString, &applicaitonpidStateMap)
		if err != nil {
			fmt.Println("1*---Unmarshalling error:", err)
		}
		fmt.Printf("1----Map read from %v: %v\n", stateFileName, applicaitonpidStateMap)
	}

	for appName, pidState := range applicaitonpidStateMap {
		//fmt.Println("1----", appName, pidState.Pid, pidState.State)
		pidExistStatus = isPid(pidState.Pid)
		fmt.Println("1----isPid() returned (", pidExistStatus, ") for PID:", pidState.Pid)
		if pidExistStatus {
			fmt.Println("1----Re-parent the applicaton:", appName, pidState.Pid)
			err = syscall.PtraceAttach(pidState.Pid)
			if err != nil {
				fmt.Println("1*---PtraceAttach error:", err)
				//break
			}

			time.Sleep(1 * time.Millisecond)
			err = syscall.PtraceCont(pidState.Pid, 0)
			//err = syscall.PtraceCont(pidState.Pid,syscall.SIGCONT)
			if err != nil {
				fmt.Println("1*---PtraceCont error:", err)
				//break
			}

			if err == nil {
				//fmt.Println("1*---Delete from deploymentMap")
				for i := 0; i < len(element); i++ {
					if element[i].CsciName == appName {
						fmt.Println("1----App added to runtime deployment map after reparenting:", appName)
						runtimedeploymentMap[pidState.Pid] = appDetailruntime{CsciName: element[i].CsciName, ListOfArguments: element[i].ListOfArguments}
						fmt.Println("1----App deleted from static deployment map after reparenting:", appName)
						element = append(element[:i], element[i+1:]...)
					}
				}
			}
		}

	}

	//=============================================================================
	fmt.Println("1----Spawning of applicatons after reparenting check")
	for _, apptospawn := range element {
		//fmt.Println("1----", apptospawn)
		fmt.Println("1----", apptospawn.CsciName)
	}

	if len(element) > 0 {
		for _, apptospawn := range element {
			fmt.Println("1----", apptospawn)
			ch, err = spawnApp(apptospawn.CsciName, apptospawn.ListOfArguments)
			if err != nil {
				fmt.Println("1----Error in spawning", err)
			}
		}
	} else {
		fmt.Println("1----No new application to spawn")
	}

	go func() {
		wg.Wait() //waitsync
		//close(ch)
	}()

	go func() {
		var ws syscall.WaitStatus
		wpid, err := syscall.Wait4(-1, &ws, syscall.WALL, nil) // -1 indicates that wait for all children
		if wpid == -1 {
			fmt.Println("6*---Error wait4() = -1 :", err, ws)
		}
		//fmt.Println("1*---Wait4 = :", wpid, err, ws)

		if _, found := runtimedeploymentMap[wpid]; found { //if not checked then both will waits ( line 181 and this one)
			fmt.Println("6----Pid present in runtimedeploymentMap,so writing to channel")
			ch <- wpid
		}
	}()

	for {
		elem, ok := <-ch
		if ok {
			fmt.Println("1----Pid of Exited applicaton:", elem)
			fmt.Println("1----Repawning the applicaton")

			appDetailruntimeTemp := runtimedeploymentMap[elem]
			nameTemp := appDetailruntimeTemp.CsciName
			argTemp := appDetailruntimeTemp.ListOfArguments
			delete(runtimedeploymentMap, elem)
			ch, err = spawnApp(nameTemp, argTemp)
			if err != nil {
				fmt.Println("1----Error in spawning", err)
			}
		}
	}
}
