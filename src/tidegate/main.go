package main

import (
	. "tidegate/core"
  "os"
//	"encoding/json"
//	"log"
//	"net/http"
)

func main() {

  
	var args = ParseArgs(os.Args[1:])
	InitLoggers(args.Verbose, args.Quiet, args.Syslog)
	RootLogger.Debugf("Connecting to %s",args.DockerDaemonAddr)
	GenerateFile(args.DockerDaemonAddr)
	
	
}
