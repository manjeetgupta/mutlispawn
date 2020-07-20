package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var redundantDeploymentMap = map[string]uint16{}
var applicaitonpidStateMap = map[string]pidState{}
var runtimedeploymentMap = map[int]appDetailruntime{}
var deploymentConfigurationMap map[string][]appDetail
var ch = make(chan int, 5)
var chStartNotifier = make(chan bool, 1)
var wg sync.WaitGroup
var stateFileName string
var arg string
var lastDigit int

var (
	// Info    :  Special Information
	Info *log.Logger
	// Warning :There is something you need to know about
	Warning *log.Logger
	// Error   :Something has failed
	Error *log.Logger
)

func init() {
	arg = os.Args[1]
	lastDigit, _ = strconv.Atoi(arg)
	lastDigit = lastDigit % 10
	logFileName := "amsm" + arg + ".log"
	file, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}

	// file can be replaced with os.Stdout or os.Stderr)

	Info = log.New(file,
		"INFO: ",
		log.Ldate|log.Ltime|log.Llongfile)

	Warning = log.New(file,
		"WARNING: ",
		log.Ldate|log.Ltime|log.Llongfile)

	Error = log.New(file,
		"ERROR: ",
		log.Ldate|log.Ltime|log.Llongfile)
}

// A Event Struct
type Event []struct {
	SubscriptionID int         `json:"id"`
	GlobalID       int         `json:"globalID"`
	Time           time.Time   `json:"time"`
	Type           string      `json:"type"`
	Data           DataDetails `json:"data"`
}

// A DataDetails Struct
type DataDetails struct {
	Action     string `json:"action"`
	Folder     string `json:"folder"`
	FolderID   string `json:"folderID"`
	Label      string `json:"label"`
	ModifiedBy string `json:"modifiedBy"`
	Path       string `json:"path"`
	Type       string `json:"type"`
}

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

	Info.Println("<>Inside readXML funtion")
	xmlFile, err := os.Open(filename)
	if err != nil {
		Error.Println("5*---", err)
	}

	Info.Println("5----Successfully opened", filename)
	defer xmlFile.Close()

	byteValue, _ := ioutil.ReadAll(xmlFile)

	var deploymentFileContent appList
	xml.Unmarshal(byteValue, &deploymentFileContent)

	/*
		for i := 0; i < len(deploymentFileContent.App); i++ {
			Info.Println("Entry: ", i)
			Info.Println("Csci_name: ", deploymentFileContent.App[i].CsciName)
			Info.Println("Airbase_id: ", deploymentFileContent.App[i].AirbaseID)
			Info.Println("Hardware_id: ", deploymentFileContent.App[i].HardwareID)
			Info.Println("Hardware_type: ", deploymentFileContent.App[i].HardwareType)
			Info.Println("Csci_id: ", deploymentFileContent.App[i].CsciID)
			Info.Println("Max_retries: ", deploymentFileContent.App[i].MaxRetries)
			Info.Println("Redundant: ", deploymentFileContent.App[i].Redundant)
			Info.Println("List_of_arguments: ", deploymentFileContent.App[i].ListOfArguments)
			Info.Println("--------------------------------------------")
		}
	*/
	Info.Println("<>Leaving readXML funtion")
	return deploymentFileContent
}

//===========================

func isPid(pid int) bool {

	Info.Println("<>Inside isPid funtion")

	if pid <= 0 {
		Error.Println("4*---Invalid pid", pid)
		return false
	}
	proc, err := os.FindProcess(int(pid))
	if err != nil {
		Error.Println("4*---", err)
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
		Error.Println("4*---", err)
		return false
	}

	switch errno {
	case syscall.ESRCH:
		return false
	case syscall.EPERM:
		return true
	}

	Info.Println("<>Leaving isPid funtion")
	return false
}

func storeMap(applicaitonpidStateMap map[string]pidState) {

	Info.Println("<>Inside storeMap funtion")
	f, err := os.Create(stateFileName)
	if err != nil {
		Error.Println("3*---", err)
		return
	}

	jsonString, err := json.Marshal(applicaitonpidStateMap)
	if err != nil {
		Error.Println("3*---", err)
		return
	}
	Info.Println("3----Marshalled Map:", string(jsonString))

	l, err := f.WriteString(string(jsonString))
	if err != nil {
		Error.Println("3*---", err)
		f.Close()
		return
	}
	Info.Printf("3----%v Bytes successfully in %v\n", l, stateFileName)

	err = f.Close()
	if err != nil {
		Error.Println("3*---", err)
		return
	}
	Info.Println("<>Leaving storeMap funtion")
}

func spawnApp(name string, arg string, flag bool) (chan int, error) {

	Info.Println("<>Inside spawnApp funtion")
	cmd := exec.Command(name, arg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout

	if err := cmd.Start(); err != nil {
		Error.Println("2*---Error in start()")
		return nil, err
	}

	runtimedeploymentMap[cmd.Process.Pid] = appDetailruntime{CsciName: name, ListOfArguments: arg}
	applicaitonpidStateMap[name] = pidState{Pid: cmd.Process.Pid, State: 0}
	Info.Println("2----Updated maps after spawning are below: ")
	Info.Println("2----", applicaitonpidStateMap)
	Info.Println("2----", runtimedeploymentMap)
	storeMap(applicaitonpidStateMap)

	// Update the redundantDeploymentMap
	if v, found := redundantDeploymentMap[name]; found {
		fmt.Printf("2----%s state before in redundantDeploymentMap :%016b\n", name, v)
		v = v ^ 1<<lastDigit
		redundantDeploymentMap[name] = v
		v, _ := redundantDeploymentMap[name]
		fmt.Printf("2----%s state after spawning in redundantDeploymentMap :%016b\n", name, v)
	}
	updateRedundantDeploymentMap()

	wg.Add(1)
	go func() {
		fmt.Println("2----Waiting for", cmd.Process.Pid)
		cmd.Wait()
		defer wg.Done()
		if _, found := runtimedeploymentMap[cmd.Process.Pid]; found { //if not checked then both will waits ( line 322 and this one)
			Info.Println("2----Pid present in runtimedeploymentMap,so writing to channel")
			ch <- cmd.Process.Pid
		}
	}()

	Info.Println("<>Leaving spawnApp funtion")
	return ch, nil

}

func initializeRedundantDeploymentMap() {
	fmt.Println("<>Inside initializeRedundantDeploymentMap funtion")

	jsonString, err := json.Marshal(redundantDeploymentMap)
	if err != nil {
		fmt.Println("7*---Marshall()", err)
		return
	}
	fmt.Println("7----Marshalled Map to save in redundantDeploymentMap.json:", string(jsonString))
	f, err := os.Create("redundantDeploymentMap.json")
	if err != nil {
		fmt.Println("7*---Create()", err)
		return
	}
	l, err := f.WriteString(string(jsonString))
	if err != nil {
		fmt.Println("7*---Write()", err)
		return
	}
	fmt.Printf("7----%v Bytes successfully in redundantdeploymentMap.json\n", l)

	err = f.Close()
	if err != nil {
		fmt.Println("7*---Close()", err)
		return
	}
	fmt.Println("<>Leaving initializeRedundantDeploymentMap funtion")
}

func updateRedundantDeploymentMap() {

	fmt.Println("<>Inside updateRedundantDeploymentMap funtion")
	jsonString, err := json.Marshal(redundantDeploymentMap)
	if err != nil {
		fmt.Println("8*---Marshall()", err)
		return
	}
	fmt.Println("8----Marshalled Map to save in redundantDeploymentMap.json:", string(jsonString))

	f, err := os.OpenFile("redundantDeploymentMap.json", os.O_WRONLY, 0755)
	if err != nil {
		fmt.Println("8*---Open()", err)
		return
	}
	l, err := f.WriteString(string(jsonString))
	if err != nil {
		fmt.Println("8*---Write()", err)
		return
	}
	fmt.Printf("8----%v Bytes successfully in redundantdeploymentMap.json\n", l)

	err = f.Close()
	if err != nil {
		fmt.Println("8*---Close()", err)
		return
	}

	fmt.Println("<>Leaving updateRedundantDeploymentMap funtion")
}

func main() {

	Info.Println("...............................................")
	Info.Println("AMSM has started on hardware ID :", arg)
	fmt.Println("AMSM has started on hardware ID :", arg)
	Info.Println("<>Inside main funtion")
	var pidExistStatus bool = false
	var deploymentFileContent appList
	deploymentFileContent = readXML("DeploymentConfiguration.xml")
	deploymentConfigurationMap = make(map[string][]appDetail)

	//Initialize redundantDeploymentMap.json file if absent
	if _, err := os.Stat("redundantDeploymentMap.json"); os.IsNotExist(err) {
		fmt.Println("1----File redundantDeploymentMap.json does not exit. So creating a new file")
		for i := 0; i < len(deploymentFileContent.App); i++ {
			if deploymentFileContent.App[i].Redundant {
				redundantDeploymentMap[deploymentFileContent.App[i].CsciName] = 0
			}
		}
		initializeRedundantDeploymentMap()
	}
	//Creating Deployment map
	for i := 0; i < len(deploymentFileContent.App); i++ {
		//Generate key
		s := strings.TrimSpace(deploymentFileContent.App[i].HardwareID)
		listOfHardware := strings.Split(s, ",")
		for j := range listOfHardware {
			hw := listOfHardware[j]
			combinedKey := strconv.Itoa(deploymentFileContent.App[i].AirbaseID) + hw
			value, ok := deploymentConfigurationMap[combinedKey]
			if ok == true {
				//Info.Println("1----",combinedKey,":key already present in map")
				value = append(value, deploymentFileContent.App[i])
				deploymentConfigurationMap[combinedKey] = value
			} else {
				//Info.Println("1----",combinedKey,":new key")
				var value []appDetail
				value = append(value, deploymentFileContent.App[i])
				deploymentConfigurationMap[combinedKey] = value
			}
		}
	}

	//Info.Println("1----Complete DeploymentMap:\n", deploymentConfigurationMap)
	Info.Println("1----Hardware", arg, "is configured for following applications:")
	element := deploymentConfigurationMap["1"+arg]
	for _, apptospawn := range element {
		Info.Println("1----", apptospawn)
		//Info.Println("1----", apptospawn.CsciName)
	}

	//Read the Redundant map in memeory
	jsonFile, err := os.Open("redundantDeploymentMap.json")
	defer jsonFile.Close()
	if err != nil {
		fmt.Println("1*---", err)
	} else {
		jsonString, _ := ioutil.ReadAll(jsonFile)
		err = json.Unmarshal(jsonString, &redundantDeploymentMap)
		if err != nil {
			fmt.Println("1*---Unmarshalling error:", err)
		}
		fmt.Printf("1----Map read from redundantDeploymentMap.json: %v\n", redundantDeploymentMap)
	}

	//Read the ClientState map in memeory
	stateFileName = "ClientState" + arg + ".json"
	jsonFile, err = os.Open(stateFileName)
	defer jsonFile.Close()
	if err != nil {
		Error.Println("1*---", err)
	} else {
		jsonString, _ := ioutil.ReadAll(jsonFile)
		err = json.Unmarshal(jsonString, &applicaitonpidStateMap)
		if err != nil {
			Error.Println("1*---Unmarshalling error:", err)
		}
		Info.Printf("1----Map read from %v: %v\n", stateFileName, applicaitonpidStateMap)
	}

	//Reparenting Check
	for appName, pidState := range applicaitonpidStateMap {
		pidExistStatus = isPid(pidState.Pid)
		Info.Println("1----isPid() returned (", pidExistStatus, ") for PID:", pidState.Pid)
		if pidExistStatus {
			Info.Println("1----Re-parent the applicaton:", appName, pidState.Pid)
			err = syscall.PtraceAttach(pidState.Pid)
			if err != nil {
				Error.Println("1*---PtraceAttach error:", err)
			}
			var ws syscall.WaitStatus
			_, err := syscall.Wait4(pidState.Pid, &ws, syscall.WSTOPPED, nil)
			if err != nil {
				Error.Printf("1*---Error waiting after ptrace attach in pid %d :%v\n", pidState.Pid, err)
			}

			if ws.Exited() {
				Error.Println("1*---Exited:Normal termination after PtraceAttach")
			}
			if ws.Signaled() {
				Error.Println("1*---Signaled:Abnormal termination after PtraceAttach")
			}
			if ws.Continued() {
				Error.Println("1*---Continued after PtraceAttach")
			}
			if ws.CoreDump() {
				Error.Println("1*---CoreDump after PtraceAttach")
			}
			if ws.Stopped() {
				//time.Sleep(10 * time.Millisecond)
				Info.Println("1----Stop signal after PtraceAttach : ", ws.StopSignal())
				err = syscall.PtraceCont(pidState.Pid, 0)
				if err != nil {
					Error.Println("1*---PtraceCont error:", err, pidState.Pid)
				}

				var ws syscall.WaitStatus
				_, err := syscall.Wait4(pidState.Pid, &ws, syscall.WNOHANG, nil)
				if err != nil {
					Error.Printf("1*---Error waiting after ptrace continue in pid %d :%v\n", pidState.Pid, err)
				}

				if ws.Exited() {
					Info.Println("1----Exited:Normal termination after PtraceCont")
				}
				if ws.Signaled() {
					Error.Println("1*---Signaled:Abnormal termination after PtraceCont")
				}
				if ws.Continued() {
					Error.Println("1*---Continued after PtraceCont")
				}
				if ws.CoreDump() {
					Error.Println("1*---CoreDump after PtraceCont")
				}
				if ws.Stopped() {
					Error.Println("1*---Stopped after PtraceCont")
				}

				go func() {
					var ws syscall.WaitStatus
					wpid, err := syscall.Wait4(-1, &ws, syscall.WSTOPPED, nil) // -1 indicates that wait for all children
					if wpid == -1 {
						fmt.Println("6*---Error wait4() = -1 :", err, ws)
					}
					if _, found := runtimedeploymentMap[wpid]; found { //if not checked then both will waits ( line 181 and this one)
						fmt.Println("6----Pid present in runtimedeploymentMap,so writing to channel")
						syscall.Kill(wpid, syscall.SIGKILL)
						ch <- wpid
					}
				}()
			}

			if err == nil {
				Info.Println("1----PtraceCont successful")
				for i := 0; i < len(element); i++ {
					if element[i].CsciName == appName {
						Info.Println("1----App added to runtime deployment map after reparenting:", appName)
						runtimedeploymentMap[pidState.Pid] = appDetailruntime{CsciName: element[i].CsciName, ListOfArguments: element[i].ListOfArguments}
						Info.Println("1----App deleted from static deployment map after reparenting:", appName)
						element = append(element[:i], element[i+1:]...)
					}
				}
			}
		}

	}

	//Spawning of applicatons
	Info.Println("1----Spawning of applicatons after reparenting check")
	for _, apptospawn := range element {
		Info.Println("1----", apptospawn.CsciName)
	}

	if len(element) > 0 {
		for _, apptospawn := range element {
			Info.Println("1----", apptospawn)
			ch, err = spawnApp(apptospawn.CsciName, apptospawn.ListOfArguments, true)
			if err != nil {
				Error.Println("1----Error in spawning", err)
			}
		}
	} else {
		Info.Println("1----No new application to spawn")
	}

	//Wait  for sync group
	go func() {
		wg.Wait()
		//close(ch)
	}()

	chStartNotifier <- true
	//This should become active only after Initialzaton is complete.
	go fileChangeNotifier()

	//Infinite loop to read from channel
	for {
		elem, ok := <-ch
		if ok {
			Info.Println("1----Pid of Exited applicaton:", elem)
			Info.Println("1----Repawning the applicaton")

			appDetailruntimeTemp := runtimedeploymentMap[elem]
			nameTemp := appDetailruntimeTemp.CsciName
			argTemp := appDetailruntimeTemp.ListOfArguments
			delete(runtimedeploymentMap, elem)
			if v, found := redundantDeploymentMap[nameTemp]; found {
				fmt.Printf("1----%s state after crash in redundantDeploymentMap :%016b\n", nameTemp, v)
				v = v ^ 1<<lastDigit
				redundantDeploymentMap[nameTemp] = v
				v, _ := redundantDeploymentMap[nameTemp]
				fmt.Printf("1----%s old state has been reset in redundantDeploymentMap :%016b\n", nameTemp, v)
				updateRedundantDeploymentMap()
			}

			ch, err = spawnApp(nameTemp, argTemp, false)
			if err != nil {
				Warning.Println("1----Error in spawning", err)
			}
		}
	}
}

func fileChangeNotifier() {
	fmt.Println("<>Inside fileChangeNotifier funtion")

	var mostRecentID int
	var mostRecentIDstr string

	startFor := <-chStartNotifier
	fmt.Println("startFor --->", startFor)
	if startFor {
		for {
			client := &http.Client{}
			req, err := http.NewRequest("GET", "http://localhost:8384/rest/events?events=LocalChangeDetected", nil)
			if err != nil {
				log.Fatal("Error Reading request", err)
			}

			req.Header.Set("X-API-Key", "crSRLDA6wkz75FuSYoeMtCcdoVi4tsJg")
			q := req.URL.Query()
			q.Add("since", mostRecentIDstr)

			req.URL.RawQuery = q.Encode()

			resp, err := client.Do(req)
			if err != nil {
				log.Fatal("Error Reading response", err)
			}

			if resp.Body != nil {
				defer resp.Body.Close()
			}

			responseBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Fatal("Error Reading body", err)
			}

			var responseObject Event
			json.Unmarshal(responseBody, &responseObject)
			fmt.Println(string(responseBody))
			fmt.Println("Length of Response:", len(responseObject))

			if len(responseObject) > 0 {
				for i := 0; i < len(responseObject); i++ {
					fmt.Println("ID:", responseObject[i].SubscriptionID)
					// 	fmt.Println("GlobalID:", responseObject[i].GlobalID)
					// 	fmt.Println("Time:", responseObject[i].Time)
					// 	fmt.Println("Type:", responseObject[i].Type)
					// 	fmt.Println("Data->Action:", responseObject[i].Data.Action)
					// 	fmt.Println("Data->Folder:", responseObject[i].Data.Folder)
					// 	fmt.Println("Data->FolderID:", responseObject[i].Data.FolderID)
					// 	fmt.Println("Data->Label:", responseObject[i].Data.Label)
					// 	fmt.Println("Data->ModifiedBy:", responseObject[i].Data.ModifiedBy)
					// 	fmt.Println("Data->Path:", responseObject[i].Data.Path)
					// 	fmt.Println("Data->Type:", responseObject[i].Data.Type)
				}
				mostRecentID = responseObject[len(responseObject)-1].SubscriptionID
				mostRecentIDstr = strconv.Itoa(mostRecentID)
			}
			fmt.Println("In for..")
			time.Sleep(100 * time.Millisecond)
		}
	}
	fmt.Println("<>Leaving fileChangeNotifier funtion")
}
