package main

import (
	//	"github.com/aacebedo/tidegate/src/core"
	//	"github.com/aacebedo/tidegate/src/monitors"
	//	"github.com/aacebedo/tidegate/src/backends"
	"crypto/tls"
	"github.com/aacebedo/tidegate/src/core"
	"github.com/op/go-logging"
	"log"
	"net/http"
	"net/url"
	"fmt"
	"bufio"
	"os"
	"net"
	"time"
)

var logger = logging.MustGetLogger("tidegate")

type tcpKeepAliveListener struct {
        *net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
        tc, err := ln.AcceptTCP()
        if err != nil {
                return
        }
        tc.SetKeepAlive(true)
        tc.SetKeepAlivePeriod(3 * time.Minute)
        return tc, nil
}


func main() {
	proxy := core.NewMultipleHostReverseProxy(&url.URL{
		Scheme: "http",
		Host:   "localhost:8086",
	})
	cfg := &tls.Config{}

	cert, err := tls.LoadX509KeyPair("/etc/letsencrypt/live/popular-design.fr/fullchain.pem", 
	                                  "/etc/letsencrypt/live/popular-design.fr/privkey.pem")
	if err != nil {
		log.Fatal(err)
	}
	cfg.Certificates = append(cfg.Certificates, cert)
	

	server := http.Server{
		Addr:      "localhost:9090",
		Handler:   proxy,
		TLSConfig: cfg,
	}



        ln, err := net.Listen("tcp", server.Addr)
        if err != nil {
                log.Fatal("error")
        }

        tlsListener := tls.NewListener(tcpKeepAliveListener{ln.(*net.TCPListener)}, cfg)
        go server.Serve(tlsListener)



	//go server.ListenAndServeTLS("", "")
  
  reader := bufio.NewReader(os.Stdin)
fmt.Print("Enter text: ")
text, _ := reader.ReadString('\n')
fmt.Println(text)

cert, err = tls.LoadX509KeyPair("/etc/letsencrypt/live/acebedo.fr/fullchain.pem", 
	                                  "/etc/letsencrypt/live/acebedo.fr/privkey.pem")
	if err != nil {
		log.Fatal(err)
	}
	cfg.Certificates = append(cfg.Certificates, cert)

	// keep adding remaining certs to cfg.Certificates

	cfg.BuildNameToCertificate()


fmt.Println("Wait next call")
text, _ = reader.ReadString('\n')
fmt.Println(text)

	//	serverManager := core.NewServerManager()
	//
	//	var args = core.ParseArgs(os.Args[1:])
	//	core.InitLoggers(args.Verbose, args.Quiet, args.Syslog)
	//	//rpBackend, _ := backends.NewNGINXBackend("./", "/usr/sbin")
	//	//serverStorage.AddServerObserver(backend)
	//	dockerMgr, err := monitors.NewDockerManager(args.DockerDaemonAddr)
	//	dockerMgr.AddMonitor(serverManager)
	//	certBackend := backends.NewLetsEncryptBackend()
	//
	//	//ServerManager.Observe(DockerEventMonitor)
	//	//serverManager.AddServerMonitor(rpBackend)
	//	//serverManager.AddMonitor(certBackend)
	//	serverManager.AddServerMonitor(rpBackend)

	//
	//	//err := backend.Start()
	//	if err == nil {
	//		dockerMgr.Start()
	//		dockerMgr.Join()
	//	} else {
	//		//core.logger.Fatalf("Unable to start backend: %s", err.Error())
	//	}
	//
	//	//	client, _ = dockerclient.NewDockerClient(args.DockerDaemonAddr, nil)
	//	//
	//	//	//client.StartMonitorEvents(eventCallback, nil)
	//	//
	//	//	var containers, err = client.ListContainers(false, false, "")
	//	//	if err != nil {
	//	//		RootLogger.Fatalf("Unable to connect to docker daemon on '%s'. Are you sure the daemon is running ?", args.DockerDaemonAddr)
	//	//	}
	//	//	for _, c := range containers {
	//	//		ProcessContainer(&c)
	//	//	}
	//	//
	//	//	backend, err := NewNGINXBackend("./bin/", "/usr/sbin")
	//	//	if err == nil {
	//	//		RootLogger.Debugf("NGINX Backend successfully created")
	//	//	} else {
	//	//		RootLogger.Warningf("Unable to create NGINX Backend: %s", err.Error())
	//	//		return
	//	//	}
	//	//
	//	//	backend.Start()
	//	//
	//	//	for server := range servers.Iter() {
	//	//		genErr := backend.HandleServerCreation(server.(*Server))
	//	//		if genErr != nil {
	//	//			RootLogger.Warningf("Unable to handle server creation: %s", genErr.Error())
	//	//		}
	//	//	}
	//	//
	//	//	//daemon := NGINXDaemon{ConfigPath:"/home/aacebedo/Seafile/Private/workspace/tidegate_go/bin",BinPath:"/usr/sbin"}
	//	//	//daemon.Start()
	//	//	//daemon.Stop()
	//	//
	//	//	waitForInterrupt()

}
