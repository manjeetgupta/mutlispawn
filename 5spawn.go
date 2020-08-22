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
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/sparrc/go-ping"
)

var nodeConnectionMap = map[string]nodeData{}
var redundantDeploymentMap = map[string]uint16{}
var applicaitonpidStateMap = map[string]pidState{}
var runtimedeploymentMap = map[int]appDetailruntime{}
var deploymentConfigurationMap map[string][]appDetail
var ch = make(chan int, 5)
var chStartFileNotifier = make(chan bool, 1)
var chStartStateResponder = make(chan bool, 1)
var chStartNodeDisconnectNotifier = make(chan bool, 1)
var chStartNodeReConnectNotifier = make(chan int, 1)
var wg sync.WaitGroup
var stateFileName string
var arg string
var gatewayIP string
var selfID string
var lastDigit int
var selfIsolationCount int

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
	gatewayIP = os.Args[2]
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

//Tag:9
func readRedundantDeploymentMapFile() {
	Info.Println("<>Inside readRedundantDeploymentMapFile funtion(9)")
	jsonFile, err := os.Open("redundantDeploymentMap.json")
	defer jsonFile.Close()

	if err != nil {
		Error.Println("9*---File open error:", err)
	} else {
		jsonString, _ := ioutil.ReadAll(jsonFile)
		err = json.Unmarshal(jsonString, &redundantDeploymentMap)
		if err != nil {
			Error.Println("9*---Unmarshalling error:", err)
			return
		}
		Info.Printf("9----Map read from redundantDeploymentMap.json: %v\n", redundantDeploymentMap)
		fmt.Printf("9----Map read from redundantDeploymentMap.json: %v\n", redundantDeploymentMap)
	}
	Info.Println("<>Leaving readRedundantDeploymentMapFile funtion(9)")
}

//Tag:8
func readClientStateMapFile() {
	Info.Println("<>Inside readClientStateMapFile funtion(8)")
	stateFileName = "clientState" + arg + ".json"
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
	Info.Println("<>Leaving readClientStateMapFile funtion(8)")
}

//Tag:7
func initializeRedundantDeploymentMap() {
	Info.Println("<>Inside initializeRedundantDeploymentMap funtion(7)")
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
	Info.Println("<>Leaving initializeRedundantDeploymentMap funtion(7)")
}

//Tag:6
func updateRedundantDeploymentMap() {
	Info.Println("<>Inside updateRedundantDeploymentMap funtion(6)")
	f, err := os.Create("redundantDeploymentMap.json")
	defer f.Close()
	if err != nil {
		Error.Println("6*---Open()", err)
		return
	}

	jsonString, err := json.Marshal(redundantDeploymentMap)
	if err != nil {
		Error.Println("6*---Marshall()", err)
		return
	}
	Info.Println("6----Marshalled Map going to be saved in redundantDeploymentMap.json:", string(jsonString))

	l, err := f.WriteString(string(jsonString))
	if err != nil {
		Error.Println("6*---Write()", err)
		return
	}
	Info.Printf("6----%v Bytes successfully in redundantdeploymentMap.json\n", l)

	err = f.Close()
	if err != nil {
		Error.Println("6*---Close()", err)
		return
	}

	Info.Println("<>Leaving updateRedundantDeploymentMap funtion(6)")
}

//Tag:5
func readXML(filename string) appList {
	Info.Println("<>Inside readXML funtion(5)")
	xmlFile, err := os.Open(filename)
	if err != nil {
		Error.Println("5*---", err)
	}

	Info.Println("5----Successfully opened", filename)
	defer xmlFile.Close()

	byteValue, _ := ioutil.ReadAll(xmlFile)

	var deploymentFileContent appList
	xml.Unmarshal(byteValue, &deploymentFileContent)

	/* // Print the contents of Deployment xml.
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
	Info.Println("<>Leaving readXML funtion(5)")
	return deploymentFileContent
}

//Tag:4
func isPid(pid int) bool {
	Info.Println("<>Inside isPid funtion(4)")

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

	Info.Println("<>Leaving isPid funtion(4)")
	return false
}

//Tag:3
func storeMap(applicaitonpidStateMap map[string]pidState) {
	Info.Println("<>Inside storeMap funtion(3)")
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
	Info.Println("<>Leaving storeMap funtion(3)")
}

//Tag:2
func spawnApp(name string, arg string, flag bool, redundant bool) (chan int, error) {

	Info.Println("<>Inside spawnApp funtion(2)")
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
			Info.Printf("2----%s state before in redundantDeploymentMap :%016b\n", name, v)
			if v&(1<<lastDigit) == 0 { //Initialization
				Info.Println("2----Initializaton case :")
				v = v | (1 << lastDigit)
				//is there active in the system
				if v&65280 == 0 {
					//There is no active in the system
					statebit := lastDigit + 8
					v = v | (1 << statebit)
					//update map
				}
			} else { //on crash active multiple instance
				Info.Println("2----Active crash multiple instance case :")
				//v = v | 1<<lastDigit
				statebit := lastDigit + 8
				v = v & ^(1 << statebit)
				//Find 1 in other half except lastDigit.
				vv := v
				vv = vv &^ (1 << lastDigit)
				Info.Printf("2----%s state before searching for other node in redundantDeploymentMap :%016b\n", name, vv)
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
			Info.Printf("2----%s state after spawning in redundantDeploymentMap :%016b\n", name, v)
		}
		updateRedundantDeploymentMap()
	}

	wg.Add(1)

	//Tag:G1
	go func() {
		Info.Println("G1----Waiting for", cmd.Process.Pid)
		cmd.Wait()
		defer wg.Done()
		if _, found := runtimedeploymentMap[cmd.Process.Pid]; found { //if not checked then both will waits ( line 322 and this one)
			Info.Println("G1----Pid present in runtimedeploymentMap,so writing to channel")
			ch <- cmd.Process.Pid
		}
	}()

	Info.Println("<>Leaving spawnApp funtion(2)")
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
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGSEGV, syscall.SIGABRT)

	//Sleep to tune up the sync service
	Info.Println("1----Waiting for service to tune in.....")
	time.Sleep(5 * time.Second)

	//Initialize redundantDeploymentMap.json file if absent
	if _, err := os.Stat("redundantDeploymentMap.json"); os.IsNotExist(err) {
		Info.Println("1----File redundantDeploymentMap.json does not exit. So creating a new file")
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
				//Info.Println("1----", combinedKey, ":key already present in map")
				value = append(value, deploymentFileContent.App[i])
				deploymentConfigurationMap[combinedKey] = value
			} else {
				//Info.Println("1----", combinedKey, ":new key")
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

	//Read the ClientState map in memeory
	readClientStateMapFile()

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
						Error.Println("G2*---Error wait4() = -1 :", err, ws)
					}
					if _, found := runtimedeploymentMap[wpid]; found { //if not checked then both will waits ( line 181 and this one)
						Error.Println("G2----Pid present in runtimedeploymentMap,so writing to channel")
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

	//Tag:G3
	go func() {
		wg.Wait() //Wait  for sync group
		//close(ch)
	}()

	//Retrieving information from SYNC to create map @Initialization
	createNodeConnectionMap()
	Info.Println("1----NodeConnnectionMap @Initilization:")
	Info.Println("1----:", nodeConnectionMap)

	//This should become active only after Initialzaton is complete and sync is running.
	go fileChangeNotifier()

	go stateResponder()

	go nodeConnectionNotifier()

	go nodeDisconnectionNotifier()

	//Graceful exit
	//Tag:G8
	go func() {
		sig := <-sigs
		Info.Println("G8----Graceful exit started:", sig)
		Info.Println("G8----Node removal triggered:")
		triggerNodeRemovalSelf()
		os.Exit(0)
	}()

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
					Error.Println("1*---Inside crash:", err)
				} else { //Read redunadant.json
					jsonString, _ := ioutil.ReadAll(jsonFile)
					err = json.Unmarshal(jsonString, &redundantDeploymentMap)
					if err != nil {
						Error.Println("1*---Inside crash:Unmarshalling error:", err)
					}
					Info.Printf("1----Inside crash:Map read from redundantDeploymentMap.json: %v\n", redundantDeploymentMap)
				}

				if v, found := redundantDeploymentMap[nameTemp]; found {
					Info.Printf("1----Inside crash:%s state just after crash in redundantDeploymentMap :%016b\n", nameTemp, v)
					statebit := lastDigit + 8

					if v&(1<<statebit) == 0 {
						Info.Println("1----Inside crash:Passive instance crashed")
						ch, err = spawnApp(nameTemp, argTemp, false, true)
					} else {

						//Check whether it is single instance or multiple
						vv := v
						vv = vv &^ (1 << lastDigit)
						vv = vv &^ (1 << statebit)

						if vv == 0 {
							Info.Println("1----Inside crash:Active instance crashed, Single Instance")
							ch, err = spawnApp(nameTemp, argTemp, false, true)
						} else {
							Info.Println("1----Inside crash:Active instance crashed, Multiple Instance")
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
} //END of MAIN

//Tag:G4
func fileChangeNotifier() {
	Info.Println("<>Inside fileChangeNotifier funtion(G4)")

	var mostRecentID int
	var mostRecentIDstr string

	startFor := <-chStartFileNotifier
	//fmt.Println("G4----startFor recieved:", startFor)
	if startFor {
		for {
			client := &http.Client{}
			req, err := http.NewRequest("GET", "http://localhost:8384/rest/events?events=RemoteChangeDetected", nil)
			//req, err := http.NewRequest("GET", "http://localhost:8384/rest/events?events=ItemFinished", nil)
			if err != nil {
				Error.Println("*G4---Error Creating request", err)
			}

			req.Header.Set("X-API-Key", "manjeettest")
			q := req.URL.Query()
			q.Add("since", mostRecentIDstr)

			req.URL.RawQuery = q.Encode()

			resp, err := client.Do(req)
			if err != nil {
				Error.Println("*G4---Error Reading response.", err)
			}

			if resp.Body != nil {
				defer resp.Body.Close()
			}

			responseBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				Error.Println("*G4---Error Reading body", err)
			}

			var responseObject Event
			json.Unmarshal(responseBody, &responseObject)
			//Info.Println("G4----Length of response:", len(responseObject))

			if len(responseObject) > 0 {
				//Read the file and implement logic for pushing the state change to apps in long polling REST API.TODO
				//Set its status bit whenever a node came after disconnection.
				Info.Println("G4----Response recieved:", string(responseBody))
				fmt.Println("G4----Redunadant map after RemoteChangeDetected event")
				readRedundantDeploymentMapFile()

				if selfIsolationCount > 0 {
					selfIsolationCount = 0

					fmt.Println("G4----Node has reconnected after disconnection")
					go func() {
						var exitLoop bool = false
						for {
							//Read connection map for connectionstatus of other nodes
							for _, element := range nodeConnectionMap {
								if element.Address != "SELF" && element.ConnectionStatus >= 7 {
									for key, v := range redundantDeploymentMap {
										c1 := v & (1 << lastDigit)
										if c1 == 0 {
											//Is key present in deployment map of this node
											element := deploymentConfigurationMap["1"+arg]
											for _, apptospawn := range element {
												if key == apptospawn.CsciName {
													fmt.Println("G4----Setting status bit after reconnection in the required application")
													fmt.Printf("G4----%s Status bit present in redundantDeploymentMap before reconnection :%016b\n", key, v)
													v = v | (1 << lastDigit)
													fmt.Printf("G4----%s Status bit present in redundantDeploymentMap after reconnection :%016b\n", key, v)
													redundantDeploymentMap[key] = v
													updateRedundantDeploymentMap()
													break
												}
											}
										}
									}
									exitLoop = true
								}
							}

							if exitLoop {
								break
							} else {
								time.Sleep(1 * time.Second)
							}
						}
					}()
				}

				for i := 0; i < len(responseObject); i++ {
					//  fmt.Println("G4---ID:", responseObject[i].SubscriptionID)
					// 	fmt.Println("GlobalID:", responseObject[i].GlobalID)
					// 	fmt.Println("Time:", responseObject[i].Time)
					// 	fmt.Println("Type:", responseObject[i].Type)
					// 	fmt.Println("Data->Action:", responseObject[i].Data.Action)
					// 	fmt.Println("Data->Folder:", responseObject[i].Data.Folder)
					// 	fmt.Println("Data->FolderID:", responseObject[i].Data.FolderID)
					// 	fmt.Println("Data->Label:", responseObject[i].Data.Label)
					//  fmt.Println("Data->ModifiedBy:", responseObject[i].Data.ModifiedBy)
					// 	fmt.Println("Data->Path:", responseObject[i].Data.Path)
					// 	fmt.Println("Data->Type:", responseObject[i].Data.Type)
				}
				mostRecentID = responseObject[len(responseObject)-1].SubscriptionID
				mostRecentIDstr = strconv.Itoa(mostRecentID)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	Info.Println("<>Leaving fileChangeNotifier funtion(G4)")
}

//Tag:G5
func stateResponder() {
	Info.Println("<>Inside stateResponder funtion(G5)")
	startFor := <-chStartStateResponder
	if startFor {

		http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
			fmt.Fprint(res, "Improper address:Use /state?name=<NAME>")
		})

		http.HandleFunc("/state", func(res http.ResponseWriter, req *http.Request) {
			keys, ok := req.URL.Query()["name"]

			if !ok || len(keys[0]) < 1 {
				Error.Println("*G5---Url Param 'name' is missing")
				return
			}

			key := keys[0]
			Info.Println("G5----State requested for : " + string(key))
			//Retrive the state from json.
			readRedundantDeploymentMapFile()

			if v, found := redundantDeploymentMap[key]; found {
				Info.Printf("G5----%s State present in redundantDeploymentMap :%016b\n", key, v)
				//vv := string(v)
				statebit := lastDigit + 8
				//fmt.Println("G3statebit==", statebit)
				if v&(1<<statebit) == 0 {
					Info.Println("G5----Server replied 0")
					fmt.Fprint(res, "0")
				} else {
					Info.Println("G5----Server replied 1")
					fmt.Fprint(res, "1")
				}

			}

		})

		http.ListenAndServe(":9000", nil)
	}
	Info.Println("<>Leaving stateResponder funtion(G5)")
}

//Tag:G6
func nodeDisconnectionNotifier() {
	Info.Println("<>Inside nodeDisconnectionNotifier funtion(G6)")
	startFor := <-chStartNodeDisconnectNotifier
	ticker := time.NewTicker(10 * time.Second)

	if startFor { //Wait for trigger
		for _ = range ticker.C {
			Info.Println("G6----Ticker entered")
			for key, element := range nodeConnectionMap { //Iterate map every 10 sec
				if element.Address != "SELF" {
					//if element.ConnectionStatus {
					Info.Println(element.HardwareID, " :Node Connection status:", element.ConnectionStatus)
					pingresult := pingtest(element.Address)

					if pingresult == false {
						//Increment count and then reset (1-5)
						element.ConnectionStatus++
						if element.ConnectionStatus > 5 {
							element.ConnectionStatus = 1
						}
						nodeConnectionMap[key] = element
						Info.Println("G6----Checking with Gateway")
						pingresult = pingtest(gatewayIP) //Check with Gateway
						if pingresult == true {

							Info.Println("G6----key:", key, "=>", element.HardwareID, element.Address)
							Info.Println("G6----Device disconecion is confirmed as PING failed")

							lastDigittemp, _ := strconv.Atoi(element.HardwareID)
							//delete(nodeConnectionMap, mapKey)
							Info.Printf("G6----Node Map:%v\n", nodeConnectionMap)
							lastDigittemp %= 10
							Info.Println("G6----Last Digit:", lastDigittemp)

							//Diconnected node may have multiple apps,so scan whole map
							for key, v := range redundantDeploymentMap {
								Info.Printf("G6----%s==>%016b\n", key, v)
								statebittemp := lastDigittemp + 8
								c1 := v & (1 << lastDigittemp)
								c2 := v & (1 << statebittemp)
								//fmt.Println("G6----c1:", c1, "---c2", c2)
								if c1 > 0 && c2 == 0 { //Passive
									Info.Println("G6----Passive Case:")
									v = v &^ (1 << lastDigittemp)
									redundantDeploymentMap[key] = v
									Info.Printf("G6----Bitmap after Change: %s==>%016b\n", key, v)
									//break
								} else if c1 > 0 && c2 > 1 { //Active
									Info.Println("G6----Active Case:")
									v = v &^ (1 << lastDigittemp)
									v = v &^ (1 << statebittemp)
									vv := v
									Info.Printf("G6----%s Bitmap before searching for other node in redundantDeploymentMap :%016b\n", key, v)
									for i := 0; i < 8; i++ {
										if vv&1 == 1 {
											statebit := i + 8
											v = v | (1 << statebit)
										}
										vv = vv >> 1
									}
									redundantDeploymentMap[key] = v
									Info.Printf("G6----Bitmap after Change: %s==>%016b\n", key, v)
									//break
								}
							}
							updateRedundantDeploymentMap()
						} else {
							Info.Printf("G6----This node has become self isolated:%d", selfIsolationCount)
							selfIsolationCount++

							//Turn itself into BLANK node
							if selfIsolationCount == 1 { //For first time only
								Info.Println("G6----Last Digit:", lastDigit)
								for key, v := range redundantDeploymentMap {
									Info.Printf("G6----%s==>%016b\n", key, v)
									v = 0
									Info.Printf("G6 after self-isolation----%s==>%016b\n", key, v)
									redundantDeploymentMap[key] = v
								}
								updateRedundantDeploymentMap()
								cmd := exec.Command("/usr/bin/touch", "redundantDeploymentMap.json", "-r", "5spawn")
								err := cmd.Run()
								if err != nil {
									Error.Println("*G6----Error in touching the file:", err)
								}
							}
						}
					} else {
						//element.ConnectionStatus = true
						//Increment count and then reset (6-10)
						element.ConnectionStatus++
						if element.ConnectionStatus > 10 {
							element.ConnectionStatus = 6
						}
						nodeConnectionMap[key] = element
					}
				}
			}
			//time.Sleep(2 * time.Second)
		}
	}
	Info.Println("<>Leaving nodeDisconnectionNotifier funtion(G6)")
}

//Tag:G7
func nodeConnectionNotifier() {
	Info.Println("<>Inside nodeConnectionNotifier funtion(G7)")
	//Node connection event will not be recieved when cable is plugged out and then plugged in of any other node
	//while SYNC service is running on both nodes
	var mostRecentID int
	var mostRecentIDstr string

	startFor := true
	if startFor {
		for {
			client := &http.Client{}
			req, err := http.NewRequest("GET", "http://localhost:8384/rest/events?events=DeviceConnected", nil)
			if err != nil {
				Error.Println("*G7---Error Reading request", err)
			}

			req.Header.Set("X-API-Key", "manjeettest")
			q := req.URL.Query()
			q.Add("since", mostRecentIDstr)

			req.URL.RawQuery = q.Encode()

			resp, err := client.Do(req)
			if err != nil {
				Error.Println("*G7---Error Reading response", err)
			}

			if resp.Body != nil {
				defer resp.Body.Close()
			}

			responseBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				Error.Println("*G7---Error Reading body", err)
			}

			var responseObject EventConnect
			json.Unmarshal(responseBody, &responseObject)
			//fmt.Println("G7----Length of response:", len(responseObject))

			if len(responseObject) > 0 {
				Info.Println("G7----Response recieved:", string(responseBody))
				for i := 0; i < len(responseObject); i++ {
					mapKey := responseObject[i].Data.ID               //ID=DeviceID
					if d, found := nodeConnectionMap[mapKey]; found { //update only ID
						Info.Println("G7----Existing Node Update")
						//if d.GloabalID == -1 {
						//d.ReconnectFlag = true
						//}
						d.GloabalID = responseObject[i].GlobalID
						tempaddr := strings.Split(responseObject[i].Data.Addr, ":")
						d.Address = tempaddr[0]
						d.HardwareID = responseObject[i].Data.DeviceName
						lastDigitConnected, _ := strconv.Atoi(responseObject[i].Data.DeviceName)
						lastDigitConnected = lastDigitConnected % 10
						//d.ConnectionStatus = true
						d.ConnectionStatus = 6
						nodeConnectionMap[mapKey] = d
					} else {
						Info.Println("G7----New Node Entry")
						tempaddr := strings.Split(responseObject[i].Data.Addr, ":")
						nodeConnectionMap[mapKey] = nodeData{HardwareID: responseObject[i].Data.DeviceName,
							GloabalID: responseObject[i].GlobalID,
							Address:   tempaddr[0],
							//ConnectionStatus: true}
							ConnectionStatus: 6}
						//ReconnectFlag:    true}
					}
					// fmt.Println("G7---ID:", responseObject[i].SubscriptionID)
					// fmt.Println("GlobalID:", responseObject[i].GlobalID)
					// fmt.Println("Time:", responseObject[i].Time)
					// fmt.Println("G7---Type:", responseObject[i].Type)
					// fmt.Println("G7---Data->ID:", responseObject[i].Data.ID)z
					// fmt.Println("G7---Data->DeviceName:", responseObject[i].Data.DeviceName)
				}
				mostRecentID = responseObject[len(responseObject)-1].SubscriptionID
				mostRecentIDstr = strconv.Itoa(mostRecentID)
			}
			Info.Printf("G7----NodeConnectionMap @DeviceConnected:%v", nodeConnectionMap)
			//Print complete map
			// for key, element := range nodeConnectionMap {
			// 	fmt.Println("key:", key, "=>", element.HardwareID, element.GloabalID, element.Address)
			// }

			time.Sleep(100 * time.Millisecond)
		}
	}
	Info.Println("<>Leaving nodeConnectionNotifier funtion(G7)")
}

//Tag:10
func pingtest(ipaddress string) bool {
	Info.Println("<>Inside pingtest funtion(10)")
	var ret bool

	pinger, err := ping.NewPinger(ipaddress)
	if err != nil {
		Error.Println("*10----Pinger creation error:", err)
		return true
	}
	pinger.Count = 2
	pinger.Timeout = time.Millisecond * 500
	pinger.Interval = time.Millisecond * 333

	pinger.Run()
	stats := pinger.Statistics()
	//fmt.Println("10----Statistics", stats.PacketsSent, stats.PacketsRecv, stats.PacketLoss)

	if stats.PacketLoss > 0 {
		ret = false
	} else {
		ret = true
	}
	Info.Printf("10----Return value from pingtest on %v : %v", ipaddress, ret)
	Info.Println("<>Leaving pingtest funtion(10)")
	return ret
}

//Tag:11
func triggerNodeRemovalSelf() {
	Info.Println("<>Inside triggerNodeRemoval funtion(11)")
	Info.Println("11----Signal catched @shutdwon of service.Device removed fron current p2p cluster.Delete from nodeConnenctionMap")

	lastDigittemp := lastDigit
	//Diconnected node may have multiple apps,so scan whole map
	for key, v := range redundantDeploymentMap {
		Info.Printf("11----%s==>%016b\n", key, v)
		statebittemp := lastDigittemp + 8
		c1 := v & (1 << lastDigittemp)
		c2 := v & (1 << statebittemp)
		//fmt.Println("11----c1:", c1, "---c2", c2)
		if c1 > 0 && c2 == 0 { //Passive
			Info.Println("11----Passive Case:")
			v = v &^ (1 << lastDigittemp)
			redundantDeploymentMap[key] = v
			Info.Printf("11----Bitmap after Change: %s==>%016b\n", key, v)
			//break
		} else if c1 > 0 && c2 > 1 { //Active
			Info.Println("11----Active Case:")
			v = v &^ (1 << lastDigittemp)
			v = v &^ (1 << statebittemp)
			vv := v
			Info.Printf("11----%s Bitmap before searching for other node in redundantDeploymentMap :%016b\n", key, v)
			for i := 0; i < 8; i++ {
				if vv&1 == 1 {
					statebit := i + 8
					v = v | (1 << statebit)
				}
				vv = vv >> 1
			}
			redundantDeploymentMap[key] = v
			Info.Printf("11----Bitmap after Change: %s==>%016b\n", key, v)
			//break
		}
	}
	updateRedundantDeploymentMap()
	Info.Println("<>Leaving triggerNodeRemoval funtion(11)")
}

//Tag:12
func createNodeConnectionMap() {
	Info.Println("<>Inside createNodeConnectionMap funtion(12)")

	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://localhost:8384/rest/system/config", nil)
	if err != nil {
		Error.Println("*12---Error creating Config request", err)
	}

	req.Header.Set("X-API-Key", "manjeettest")
TRY1:
	resp, err := client.Do(req)
	if err != nil {
		Error.Println("*12---Error reading config response.", err)
		time.Sleep(3 * time.Second)
		goto TRY1

	} else {
		Info.Println("12----Server is up & running.")

		if resp.Body != nil {
			defer resp.Body.Close()
		}
		responseBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			Error.Println("*12---Error Reading config response body", err)
		}

		var configResponse AutoGeneratedConfig
		json.Unmarshal(responseBody, &configResponse)
		//Info.Println("12----Response recieved for Config:", string(responseBody))

		//Add Gateway in nodeConnection Map
		nodeConnectionMap["GATEWAY"] = nodeData{HardwareID: "GATEWAY",
			GloabalID:        -1,
			Address:          gatewayIP,
			ConnectionStatus: 6}

		//Retrieve Device ID and Device Name from config
		for i := 0; i < len(configResponse.Devices); i++ {
			//Info.Println("12----Device ID:", configResponse.Devices[i].DeviceID, "Device Name:", configResponse.Devices[i].Name)
			nodeConnectionMap[configResponse.Devices[i].DeviceID] = nodeData{HardwareID: configResponse.Devices[i].Name,
				GloabalID: -1,
				Address:   "NIL",
				//ConnectionStatus: false}
				ConnectionStatus: 1}
			//ReconnectFlag:    false}
		}

		req, err := http.NewRequest("GET", "http://localhost:8384/rest/system/connections", nil)
		if err != nil {
			Error.Println("*12---Error creating Connections request", err)
		}

		req.Header.Set("X-API-Key", "manjeettest")
	TRY2:
		resp, err := client.Do(req)
		if err != nil {
			Error.Println("*12---Error reading Connections response", err)
			time.Sleep(3 * time.Second)
			goto TRY2
		} else {
			if resp.Body != nil {
				defer resp.Body.Close()
			}
			responseBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				Error.Println("*12---Error Reading Connections response body", err)
			}
			//Info.Println("G4----Response recieved for Connections:", string(responseBody))

			var result map[string]interface{}
			json.Unmarshal(responseBody, &result)
			responseConnectionMap := result["connections"].(map[string]interface{})
			//Info.Println("G4----Connections Key inside respose object:==>\n", responseConnectionMap)

			//convert response to map
			var connectionPartOnly = map[string]connections{}
			//k=deviceID and v = map of connections type
			for k, v := range responseConnectionMap {
				//fmt.Println("type of value k", reflect.TypeOf(k))
				//fmt.Println("type of value v", reflect.TypeOf(v))
				jsonstringtemp, _ := json.Marshal(v) //convert map to json
				c := connections{}
				json.Unmarshal(jsonstringtemp, &c) //convert json to struct
				connectionPartOnly[k] = c
			}
			//Retrieve Connection status and Addreess for given Device ID from connnections
			for k, v := range connectionPartOnly {
				//Info.Println("12----Device ID:", k, "Connected:", v.Connected, "Address:", v.Address)
				var ConnectionStatustmp int

				if v.Connected == true {
					ConnectionStatustmp = 6
				} else {
					ConnectionStatustmp = 1
				}

				if d, found := nodeConnectionMap[k]; found { //update connectionstatus and address
					//d.ConnectionStatus = v.Connected
					d.ConnectionStatus = ConnectionStatustmp

					if d.HardwareID == arg {
						d.Address = "SELF"
						selfID = k
					} else {
						tempaddr := strings.Split(v.Address, ":")
						d.Address = tempaddr[0]
					}
					nodeConnectionMap[k] = d
				} else {
					tempaddr := strings.Split(v.Address, ":")
					nodeConnectionMap[k] = nodeData{HardwareID: "NIL",
						GloabalID:        -1,
						Address:          tempaddr[0],
						ConnectionStatus: ConnectionStatustmp}
					//ReconnectFlag:    false}
				}
			}
		}

		chStartFileNotifier <- true
		chStartStateResponder <- true
		chStartNodeDisconnectNotifier <- true
	}
	Info.Println("<>Leaving createNodeConnectionMap funtion(12)")
}
