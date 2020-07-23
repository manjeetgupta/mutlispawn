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
	Redundant       bool
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

//Tag:9
func readRedundantDeploymentMapFile() {

	Info.Println("<>Inside readRedundantDeploymentMapFile funtion")
	jsonFile, err := os.Open("redundantDeploymentMap.json")
	defer jsonFile.Close()

	if err != nil {
		fmt.Println("9*---File open error:", err)
	} else {
		jsonString, _ := ioutil.ReadAll(jsonFile)
		err = json.Unmarshal(jsonString, &redundantDeploymentMap)
		if err != nil {
			fmt.Println("9*---Unmarshalling error:", err)
			return
		}
		fmt.Printf("9----Map read from redundantDeploymentMap.json: %v\n", redundantDeploymentMap)
	}
	Info.Println("<>Leaving readRedundantDeploymentMapFile funtion")
}

//Tag:8
func readClientStateMapFile() {
	Info.Println("<>Inside readClientStateMapFile funtion")
	stateFileName = "ClientState" + arg + ".json"
	jsonFile, err := os.Open(stateFileName)
	defer jsonFile.Close()
	if err != nil {
		Error.Println("8*---", err)
	} else {
		jsonString, _ := ioutil.ReadAll(jsonFile)
		err = json.Unmarshal(jsonString, &applicaitonpidStateMap)
		if err != nil {
			Error.Println("8*---Unmarshalling error:", err)
			return
		}
		Info.Printf("8---Map read from %v: %v\n", stateFileName, applicaitonpidStateMap)
	}
	Info.Println("<>Leaving readClientStateMapFile funtion")
}

//Tag:7
func initializeRedundantDeploymentMap() {
	Info.Println("<>Inside initializeRedundantDeploymentMap funtion")

	jsonString, err := json.Marshal(redundantDeploymentMap)
	if err != nil {
		Error.Println("7*---Marshall()", err)
		return
	}
	Info.Println("7----Marshalled Map to be saved in redundantDeploymentMap.json @initialization:", string(jsonString))
	f, err := os.Create("redundantDeploymentMap.json")
	if err != nil {
		Error.Println("7*---Create()", err)
		return
	}
	l, err := f.WriteString(string(jsonString))
	if err != nil {
		Error.Println("7*---Write()", err)
		return
	}
	Info.Printf("7----%v Bytes successfully in redundantdeploymentMap.json\n", l)

	err = f.Close()
	if err != nil {
		Error.Println("7*---Close()", err)
		return
	}
	Info.Println("<>Leaving initializeRedundantDeploymentMap funtion")
}

//Tag:6
func updateRedundantDeploymentMap() {

	fmt.Println("<>Inside updateRedundantDeploymentMap funtion")

	//f, err := os.OpenFile("redundantDeploymentMap.json", os.O_RDWR, 0777)
	f, err := os.Create("redundantDeploymentMap.json")
	defer f.Close()
	if err != nil {
		fmt.Println("6*---Open()", err)
		return
	}

	jsonString, err := json.Marshal(redundantDeploymentMap)
	if err != nil {
		fmt.Println("6*---Marshall()", err)
		return
	}
	fmt.Println("6----Marshalled Map to be saved in redundantDeploymentMap.json:", string(jsonString))

	l, err := f.WriteString(string(jsonString))
	if err != nil {
		fmt.Println("6*---Write()", err)
		return
	}
	fmt.Printf("6----%v Bytes successfully in redundantdeploymentMap.json\n", l)

	err = f.Close()
	if err != nil {
		fmt.Println("6*---Close()", err)
		return
	}

	fmt.Println("<>Leaving updateRedundantDeploymentMap funtion")
}

//Tag:5
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

//Tag:4
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

//Tag:3
func storeMap(applicaitonpidStateMap map[string]pidState) {

	Info.Println("<>Inside storeMap funtion")
	f, err := os.Create(stateFileName)
	defer f.Close()
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

//Tag:2
func spawnApp(name string, arg string, flag bool, redundant bool) (chan int, error) {

	Info.Println("<>Inside spawnApp funtion")
	cmd := exec.Command(name, arg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout

	if err := cmd.Start(); err != nil {
		Error.Println("2*---Error in start()")
		return nil, err
	}

	runtimedeploymentMap[cmd.Process.Pid] = appDetailruntime{CsciName: name, ListOfArguments: arg, Redundant: redundant}
	applicaitonpidStateMap[name] = pidState{Pid: cmd.Process.Pid, State: 0}
	Info.Println("2----Updated maps after spawning are below: ")
	Info.Println("2----", applicaitonpidStateMap)
	Info.Println("2----", runtimedeploymentMap)
	storeMap(applicaitonpidStateMap)

	// Update the redundantDeploymentMap, if flag is true...from 2 places..on active crash with multiple ...on intialization
	if flag {
		if v, found := redundantDeploymentMap[name]; found {
			fmt.Printf("2----%s state before in redundantDeploymentMap :%016b\n", name, v)
			if v&(1<<lastDigit) == 0 { //Initialization
				fmt.Println("2----Initializaton case :")
				v = v | (1 << lastDigit)
				//is there active in the system
				if v&65280 == 0 {
					//There is no active in the system
					statebit := lastDigit + 8
					v = v | (1 << statebit)
					//update map
				}
			} else { //on crash active multiple instance
				fmt.Println("2----Active crash multiple instance case :")
				//v = v | 1<<lastDigit
				statebit := lastDigit + 8
				v = v & ^(1 << statebit)
				//Find 1 in other half except lastDigit.
				vv := v
				vv = vv &^ (1 << lastDigit)
				fmt.Printf("2----%s state before searching for other node in redundantDeploymentMap :%016b\n", name, vv)
				for i := 0; i < 8; i++ {
					if vv&1 == 1 {
						statebit = i + 8
						v = v | (1 << statebit)
					}
					vv = vv >> 1
				}

			}

			redundantDeploymentMap[name] = v
			v, _ := redundantDeploymentMap[name]
			fmt.Printf("2----%s state after spawning in redundantDeploymentMap :%016b\n", name, v)
		}
		updateRedundantDeploymentMap()
	}

	wg.Add(1)
	//Tag:G1
	go func() {
		fmt.Println("G1----Waiting for", cmd.Process.Pid)
		cmd.Wait()
		defer wg.Done()
		if _, found := runtimedeploymentMap[cmd.Process.Pid]; found { //if not checked then both will waits ( line 322 and this one)
			fmt.Println("G1----Pid present in runtimedeploymentMap,so writing to channel")
			ch <- cmd.Process.Pid
		}
	}()

	Info.Println("<>Leaving spawnApp funtion")
	return ch, nil

}

//Tag:1
func main() {

	Info.Println("...............................................")
	Info.Println("AMSM has started on hardware ID :", arg)
	fmt.Println("AMSM has started on hardware ID :", arg)
	Info.Println("<>Inside main funtion")
	var pidExistStatus bool = false
	var deploymentFileContent appList
	deploymentFileContent = readXML("DeploymentConfiguration.xml")
	deploymentConfigurationMap = make(map[string][]appDetail)
	var err error

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
	readRedundantDeploymentMapFile()
	// jsonFile, err := os.Open("redundantDeploymentMap.json")
	// defer jsonFile.Close()
	// if err != nil {
	// 	fmt.Println("1*---", err)
	// } else {
	// 	jsonString, _ := ioutil.ReadAll(jsonFile)
	// 	err = json.Unmarshal(jsonString, &redundantDeploymentMap)
	// 	if err != nil {
	// 		fmt.Println("1*---Unmarshalling error:", err)
	// 	}
	// 	fmt.Printf("1----Map read from redundantDeploymentMap.json: %v\n", redundantDeploymentMap)
	// }

	//Read the ClientState map in memeory
	readClientStateMapFile()
	// stateFileName = "ClientState" + arg + ".json"
	// jsonFile, err = os.Open(stateFileName)
	// defer jsonFile.Close()
	// if err != nil {
	// 	Error.Println("1*---", err)
	// } else {
	// 	jsonString, _ := ioutil.ReadAll(jsonFile)
	// 	err = json.Unmarshal(jsonString, &applicaitonpidStateMap)
	// 	if err != nil {
	// 		Error.Println("1*---Unmarshalling error:", err)
	// 	}
	// 	Info.Printf("1----Map read from %v: %v\n", stateFileName, applicaitonpidStateMap)
	// }

	//Reparenting Check
	for appName, pidState := range applicaitonpidStateMap {
		//var err error
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
				//Tag:G2
				go func() {
					var ws syscall.WaitStatus
					wpid, err := syscall.Wait4(-1, &ws, syscall.WSTOPPED, nil) // -1 indicates that wait for all children
					if wpid == -1 {
						fmt.Println("G2*---Error wait4() = -1 :", err, ws)
					}
					if _, found := runtimedeploymentMap[wpid]; found { //if not checked then both will waits ( line 181 and this one)
						fmt.Println("G2----Pid present in runtimedeploymentMap,so writing to channel")
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
						runtimedeploymentMap[pidState.Pid] = appDetailruntime{CsciName: element[i].CsciName, ListOfArguments: element[i].ListOfArguments, Redundant: element[i].Redundant}
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

			if apptospawn.Redundant {
				ch, err = spawnApp(apptospawn.CsciName, apptospawn.ListOfArguments, true, true) //true = update redundantDeploymentMap.json
			} else {
				ch, err = spawnApp(apptospawn.CsciName, apptospawn.ListOfArguments, false, false)
			}

			if err != nil {
				Error.Println("1----Error in spawning", err)
			}
		}
	} else {
		Info.Println("1----No new application to spawn")
	}

	//Wait  for sync group
	//Tag:G3
	go func() {
		wg.Wait()
		//close(ch)
	}()

	chStartNotifier <- true
	//This should become active only after Initialzaton is complete.
	go fileChangeNotifier()

	go stateResponder()

	//Infinite loop to read from channel
	for {
		elem, ok := <-ch
		if ok {
			Info.Println("1----Pid of Exited applicaton:", elem)
			Info.Println("1----Repawning the applicaton")

			appDetailruntimeTemp := runtimedeploymentMap[elem]
			nameTemp := appDetailruntimeTemp.CsciName
			argTemp := appDetailruntimeTemp.ListOfArguments
			redundantTemp := appDetailruntimeTemp.Redundant
			delete(runtimedeploymentMap, elem)

			if redundantTemp { //For Redundant Apps.

				//Read map from redundantDeploymentMap.json
				jsonFile, err := os.Open("redundantDeploymentMap.json")
				defer jsonFile.Close()
				if err != nil {
					fmt.Println("1*---Inside crash:", err)
				} else { //Read redunadant.json
					jsonString, _ := ioutil.ReadAll(jsonFile)
					err = json.Unmarshal(jsonString, &redundantDeploymentMap)
					if err != nil {
						fmt.Println("1*---Inside crash:Unmarshalling error:", err)
					}
					fmt.Printf("1----Inside crash:Map read from redundantDeploymentMap.json: %v\n", redundantDeploymentMap)
				}

				if v, found := redundantDeploymentMap[nameTemp]; found {
					fmt.Printf("1----Inside crash:%s state just after crash in redundantDeploymentMap :%016b\n", nameTemp, v)
					//check its state bit in map
					//if v & 1<<statebit !=0
					// then it is set ...means active failed...else passive failed
					statebit := lastDigit + 8

					if v&(1<<statebit) == 0 {
						fmt.Println("1----Inside crash:Passive instance crashed")
						ch, err = spawnApp(nameTemp, argTemp, false, true)
					} else {

						//Check whether it is single instance or multiple
						vv := v
						vv = vv &^ (1 << lastDigit)
						vv = vv &^ (1 << statebit)

						if vv == 0 {
							fmt.Println("1----Inside crash:Active instance crashed, Single Instance")
							ch, err = spawnApp(nameTemp, argTemp, false, true)
						} else {
							fmt.Println("1----Inside crash:Active instance crashed, Multiple Instance")
							ch, err = spawnApp(nameTemp, argTemp, true, true)
						}

					}
				}
			} else { //For Non redundant Apps
				ch, err = spawnApp(nameTemp, argTemp, false, false)
			}
			if err != nil {
				Warning.Println("1----Error in spawning", err)
			}
		}
	}
}

//Tag:G4
func fileChangeNotifier() {
	fmt.Println("<>Inside fileChangeNotifier funtion")

	var mostRecentID int
	var mostRecentIDstr string

	startFor := <-chStartNotifier
	fmt.Println("G4----startFor recieved:", startFor)
	if startFor {
		for {
			client := &http.Client{}
			req, err := http.NewRequest("GET", "http://localhost:8384/rest/events?events=RemoteChangeDetected", nil)
			if err != nil {
				fmt.Println("*G4---Error Reading request", err)
			}

			req.Header.Set("X-API-Key", "manjeettest")
			q := req.URL.Query()
			q.Add("since", mostRecentIDstr)

			req.URL.RawQuery = q.Encode()

			resp, err := client.Do(req)
			if err != nil {
				fmt.Println("*G4---Error Reading response", err)
			}

			if resp.Body != nil {
				defer resp.Body.Close()
			}

			responseBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println("*G4---Error Reading body", err)
			}

			var responseObject Event
			json.Unmarshal(responseBody, &responseObject)
			fmt.Println("G4----Response recieved:", string(responseBody))
			fmt.Println("G4----Length of response:", len(responseObject))

			if len(responseObject) > 0 {
				//Read the file and implement logic. TODO
				for i := 0; i < len(responseObject); i++ {
					fmt.Println("G4---ID:", responseObject[i].SubscriptionID)
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
			fmt.Println("G4---Request completed..")
			time.Sleep(100 * time.Millisecond)
		}
	}
	fmt.Println("<>Leaving fileChangeNotifier funtion")
}

//Tag:G5
func stateResponder() {
	Info.Println("<>Inside stateResponder funtion")
	startFor := <-chStartNotifier
	fmt.Println("G5----startFor recieved:", startFor)
	if startFor {

		http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
			fmt.Fprint(res, "Improper address:Use /state?name=<NAME>")
		})

		http.HandleFunc("/state", func(res http.ResponseWriter, req *http.Request) {
			keys, ok := req.URL.Query()["name"]

			if !ok || len(keys[0]) < 1 {
				fmt.Println("*G5---Url Param 'name' is missing")
				return
			}

			key := keys[0]
			fmt.Println("G5----State requested for : " + string(key))
			//Get the state from json.
			readRedundantDeploymentMapFile()

			if v, found := redundantDeploymentMap[key]; found {
				fmt.Printf("G5----%s State present in redundantDeploymentMap :%016b\n", key, v)
				//vv := string(v)
				statebit := lastDigit + 8
				//fmt.Println("G3statebit==", statebit)
				if v&(1<<statebit) == 0 {
					fmt.Println("G5----Server replied 0")
					fmt.Fprint(res, "0")
				} else {
					fmt.Println("G5----Server replied 1")
					fmt.Fprint(res, "1")
				}

			}

		})

		http.ListenAndServe(":9000", nil)
	}
	Info.Println("<>Leaving stateResponder funtion")
}
