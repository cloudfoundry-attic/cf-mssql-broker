// +build windows

package main

import (
	"flag"
	"golang.org/x/sys/windows/svc"
	"log"
	"os"
	"path"
)

type WindowsService struct {
}

func (ws *WindowsService) Execute(args []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (svcSpecificEC bool, exitCode uint32) {
	s <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue}
	go runMain()

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

func redirectStdStreams(logDir string) error {

	if logDir == "" {
		workingDir, err := os.Getwd()
		if err != nil {
			return err
		}
		logDir = path.Join(workingDir, "logs")
	}
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		err := os.Mkdir(logDir, 0666)
		if err != nil {
			return err
		}
	}

	logFilePath := path.Join(logDir, "mssql_broker.log")

	logFile, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND|os.O_SYNC, 0660)
	if err != nil {
		return err
	}

	log.SetOutput(logFile)
	os.Stderr = logFile
	os.Stdout = logFile

	return nil
}

func main() {

	consoleMode := false
	logDir := ""
	flag.BoolVar(&consoleMode, "console", false, "Determines if application runs in console mode")
	flag.StringVar(&logDir, "logDir", "", "The directory where the logs will be stored")

	if !flag.Parsed() {
		flag.Parse()
	}

	if consoleMode {
		runMain()
	} else {
		var err error
		//setting stdout, stderr and log output
		err = redirectStdStreams(logDir)

		if err != nil {
			panic(err.Error())
		}

		ws := WindowsService{}
		run := svc.Run

		err = run("mssql_broker", &ws)
		if err != nil {
			os.Exit(1)
		}
	}
}
