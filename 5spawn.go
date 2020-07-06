package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

var applicaitonpidStateMap = map[string]pidState{}
var runtimedeploymentMap = map[int]appDetailruntime{}
var deploymentConfigurationMap map[string][]appDetail
var ch = make(chan int, 5)
var wg sync.WaitGroup
var stateFileName string
var arg string

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

func spawnApp(name string, arg string) (chan int, error) {

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

func main() {

	Info.Println("...............................................")
	Info.Println("AMSM has started on hardware ID :", arg)
	fmt.Println("AMSM has started on hardware ID :", arg)
	Info.Println("<>Inside main funtion")
	var pidExistStatus bool = false

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

	//=================================================

	stateFileName = "ClientState" + arg + ".json"
	jsonFile, err := os.Open(stateFileName)
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

	//=============================================================================
	Info.Println("1----Spawning of applicatons after reparenting check")
	for _, apptospawn := range element {
		Info.Println("1----", apptospawn.CsciName)
	}

	if len(element) > 0 {
		for _, apptospawn := range element {
			Info.Println("1----", apptospawn)
			ch, err = spawnApp(apptospawn.CsciName, apptospawn.ListOfArguments)
			if err != nil {
				Error.Println("1----Error in spawning", err)
			}
		}
	} else {
		Info.Println("1----No new application to spawn")
	}

	go func() {
		wg.Wait() //waitsync
		//close(ch)
	}()

	// go func() {
	// 	var ws syscall.WaitStatus
	// 	wpid, err := syscall.Wait4(-1, &ws, syscall.WALL, nil) // -1 indicates that wait for all children
	// 	if wpid == -1 {
	// 		Error.Println("6*---Error wait4() = -1 :", err, ws)
	// 	}
	// 	Warning.Println("6----Wait4 triggered ( PID Signal):", wpid, ws)

	// 	if _, found := runtimedeploymentMap[wpid]; found { //if not checked then both will waits ( line 181 and this one)
	// 		Info.Println("6----Pid present in runtimedeploymentMap,so writing to channel")
	// 		ch <- wpid
	// 	}
	// }()

	for {
		elem, ok := <-ch
		if ok {
			Info.Println("1----Pid of Exited applicaton:", elem)
			Info.Println("1----Repawning the applicaton")

			appDetailruntimeTemp := runtimedeploymentMap[elem]
			nameTemp := appDetailruntimeTemp.CsciName
			argTemp := appDetailruntimeTemp.ListOfArguments
			delete(runtimedeploymentMap, elem)
			ch, err = spawnApp(nameTemp, argTemp)
			if err != nil {
				Warning.Println("1----Error in spawning", err)
			}
		}
	}
}
