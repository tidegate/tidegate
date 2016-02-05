package main

import (
	"tidegate/core"
	"tidegate/core/servers"
	"tidegate/core/backends"
  "os"	
)


func main() {
  serverStorage := servers.NewServerStorage()
  
	var args = core.ParseArgs(os.Args[1:])
	core.InitLoggers(args.Verbose, args.Quiet, args.Syslog)
	backend, _ := backends.NewNGINXBackend("./", "/usr/sbin")
	serverStorage.AddServerObserver(backend.Observer)
	dockerMonitor,_ := core.NewDockerMonitor(args.DockerDaemonAddr,serverStorage)
	err := backend.Start()
	if err == nil {
    dockerMonitor.Start()
	  dockerMonitor.Join()
	} else {
	  core.RootLogger.Fatalf("Unable to start backend: %s",err.Error())
	}
	
	
	
	
//	client, _ = dockerclient.NewDockerClient(args.DockerDaemonAddr, nil)
//
//	//client.StartMonitorEvents(eventCallback, nil)
//
//	var containers, err = client.ListContainers(false, false, "")
//	if err != nil {
//		RootLogger.Fatalf("Unable to connect to docker daemon on '%s'. Are you sure the daemon is running ?", args.DockerDaemonAddr)
//	}
//	for _, c := range containers {
//		ProcessContainer(&c)
//	}
//
//	backend, err := NewNGINXBackend("./bin/", "/usr/sbin")
//	if err == nil {
//		RootLogger.Debugf("NGINX Backend successfully created")
//	} else {
//		RootLogger.Warningf("Unable to create NGINX Backend: %s", err.Error())
//		return
//	}
//
//	backend.Start()
//
//	for server := range servers.Iter() {
//		genErr := backend.HandleServerCreation(server.(*Server))
//		if genErr != nil {
//			RootLogger.Warningf("Unable to handle server creation: %s", genErr.Error())
//		}
//	}
//
//	//daemon := NGINXDaemon{ConfigPath:"/home/aacebedo/Seafile/Private/workspace/tidegate_go/bin",BinPath:"/usr/sbin"}
//	//daemon.Start()
//	//daemon.Stop()
//
//	waitForInterrupt()
	
}


