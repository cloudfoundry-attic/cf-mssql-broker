// +build windows

package main

import (
	"flag"
	"io"
	"os"
	"path"

	"golang.org/x/sys/windows/svc"
)

type WindowsService struct {
	writer io.Writer
}

func (ws *WindowsService) Execute(args []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (svcSpecificEC bool, exitCode uint32) {
	s <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue}
	go runMain(ws.writer)

loop:
	for {
		select {
		case change := <-r:
			switch change.Cmd {
			case svc.Interrogate:
				s <- change.CurrentStatus
			case svc.Stop, svc.Shutdown:
				{
					break loop
				}
			case svc.Pause:
				s <- svc.Status{State: svc.Paused, Accepts: svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue}
			case svc.Continue:
				s <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue}
			default:
				{
					break loop
				}
			}
		}
	}
	s <- svc.Status{State: svc.StopPending}
	return
}

func main() {
	var logDir = ""
	flag.StringVar(&logDir, "logDir", "", "The directory where the logs will be stored")
	var serviceName = flag.String("serviceName", "mssql_broker", "The name of the service as installed in Windows SCM")

	if !flag.Parsed() {
		flag.Parse()
	}

	interactiveMode, err := svc.IsAnInteractiveSession()
	if err != nil {
		panic(err.Error())
	}

	if interactiveMode {
		runMain(os.Stdout)
	} else {
		var err error

		if logDir == "" {
			//will default to %windir%\System32\
			workingDir, err := os.Getwd()
			if err != nil {
				panic(err.Error())
			}
			logDir = path.Join(workingDir, "logs")
		}
		if _, err := os.Stat(logDir); os.IsNotExist(err) {
			err := os.Mkdir(logDir, 0666)
			if err != nil {
				panic(err.Error())
			}
		}

		logFilePath := path.Join(logDir, "mssql_broker.log")

		logFile, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND|os.O_SYNC, 0660)
		if err != nil {
			panic(err.Error())
		}
		defer logFile.Close()

		//setting stderr & stdout
		os.Stdout = logFile
		os.Stderr = logFile

		fileWriter := NewWinFileWriter(logFile)

		ws := WindowsService{
			writer: fileWriter,
		}

		err = svc.Run(*serviceName, &ws)
		if err != nil {
			panic(err.Error())
		}
	}
}
