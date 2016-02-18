package main

import (
	//	"github.com/aacebedo/tidegate/src/core"
	"github.com/aacebedo/tidegate/src/monitors"
	//	"github.com/aacebedo/tidegate/src/backends"
	
	"github.com/aacebedo/tidegate/src/core"
	"github.com/op/go-logging"
	"os"
	"net"
	"time"
	"crypto/rsa"
	"github.com/xenolf/lego/acme"
	"crypto/rand"
	"path"
	"io/ioutil"
	"encoding/json"
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

type MyUser struct {
    Email        string
    Registration *acme.RegistrationResource
    key          *rsa.PrivateKey
}
func (u MyUser) GetEmail() string {
    return u.Email
}
func (u MyUser) GetRegistration() *acme.RegistrationResource {
    return u.Registration
}
func (u MyUser) GetPrivateKey() *rsa.PrivateKey {
    return u.key
}

func generateCerts() {
    
  const rsaKeySize = 2048
  privateKey, err := rsa.GenerateKey(rand.Reader, rsaKeySize)
  if err != nil {
      logger.Fatal(err)
  }
  myUser := MyUser{
      Email: "alexandre@acebedo.fr",
      key: privateKey,
  }

client, err := acme.NewClient("https://acme-v01.api.letsencrypt.org/directory", &myUser, rsaKeySize)
if err != nil {
  logger.Fatal(err)
}

client.SetHTTPAddress(":81")
client.SetTLSAddress(":444")
//prov, err:= acme.NewDNSProviderManual()
//if err != nil {
//    logger.Fatal(err)
//}
//client.SetChallengeProvider(acme.HTTP01, nil)

// New users will need to register; be sure to save it
reg, err := client.Register()
if err != nil {
    logger.Fatal(err)
}
myUser.Registration = reg

// The client has a URL to the current Let's Encrypt Subscriber
// Agreement. The user will need to agree to it.
err = client.AgreeToTOS()
if err != nil {
    logger.Fatal(err)
}

// The acme library takes care of completing the challenges to obtain the certificate(s).
// Of course, the hostnames must resolve to this machine or it will fail.
bundle := false
certRes, failures := client.ObtainCertificate([]string{"blog.acebedo.fr"}, bundle, nil)
if len(failures) > 0 {
    logger.Fatal(failures)
}

// Each certificate comes back with the cert bytes, the bytes of the client's
// private key, and a certificate URL. This is where you should save them to files!
//fmt.Printf("%#v\n", certificates)

  certOut := path.Join("/tmp/cert", certRes.Domain+".crt")
  privOut := path.Join("/tmp/cert", certRes.Domain+".key")
  metaOut := path.Join("/tmp/cert", certRes.Domain+".json")
	err = ioutil.WriteFile(certOut, certRes.Certificate, 0600)
	if err != nil {
		logger.Fatalf("Unable to save Certificate for domain %s\n\t%s", certRes.Domain, err.Error())
	}

	err = ioutil.WriteFile(privOut, certRes.PrivateKey, 0600)
	if err != nil {
		logger.Fatalf("Unable to save PrivateKey for domain %s\n\t%s", certRes.Domain, err.Error())
	}

	jsonBytes, err := json.MarshalIndent(certRes, "", "\t")
	if err != nil {
		logger.Fatalf("Unable to marshal CertResource for domain %s\n\t%s", certRes.Domain, err.Error())
	}

	err = ioutil.WriteFile(metaOut, jsonBytes, 0600)
	if err != nil {
	  logger.Fatalf("Unable to save CertResource for domain %s\n\t%s", certRes.Domain, err.Error())
	}
	
	
}



func main() {


	  file := ioutil.ReadFile("/tmp/cert/blog.acebedo.fr.crt")
	  certificates, resp, error := crypto.GetOCSPForCert(file)

  
  	var args = core.ParseArgs(os.Args[1:])
		core.InitLoggers(args.Verbose, args.Quiet, args.Syslog)
		dockerMgr, err := monitors.NewDockerManager(args.DockerDaemonAddr,)
		
		if err == nil {
	  	serverManager := core.NewServerManager()
	  	
		  dockerMgr.AddMonitor(serverManager)
			dockerMgr.Start()
			time.Sleep(2 * time.Second)
			//generateCerts()
			logger.Debug("Generate !")
			dockerMgr.Join()
		} else {
			logger.Fatalf("Unable to start backend: %s", err.Error())
		}
	
	
	
  //	proxy := core.NewMultipleHostReverseProxy(&url.URL{
//		Scheme: "http",
//		Host:   "localhost:8086",
//	})
//	cfg := &tls.Config{}
//
//	cert, err := tls.LoadX509KeyPair("/etc/letsencrypt/live/popular-design.fr/fullchain.pem", 
//	                                  "/etc/letsencrypt/live/popular-design.fr/privkey.pem")
//	if err != nil {
//		log.Fatal(err)
//	}
//	cfg.Certificates = append(cfg.Certificates, cert)
//	
//
//	server := http.Server{
//		Addr:      "localhost:9090",
//		Handler:   proxy,
//		TLSConfig: cfg,
//	}
//
//
//
//        ln, err := net.Listen("tcp", server.Addr)
//        if err != nil {
//                log.Fatal("error")
//        }
//
//        tlsListener := tls.NewListener(tcpKeepAliveListener{ln.(*net.TCPListener)}, cfg)
//        go server.Serve(tlsListener)
//
//
//
//	//go server.ListenAndServeTLS("", "")
//  
//  reader := bufio.NewReader(os.Stdin)
//fmt.Print("Enter text: ")
//text, _ := reader.ReadString('\n')
//fmt.Println(text)
//
//cert, err = tls.LoadX509KeyPair("/etc/letsencrypt/live/acebedo.fr/fullchain.pem", 
//	                                  "/etc/letsencrypt/live/acebedo.fr/privkey.pem")
//	if err != nil {
//		log.Fatal(err)
//	}
//	cfg.Certificates = append(cfg.Certificates, cert)
//
//	// keep adding remaining certs to cfg.Certificates
//
//	cfg.BuildNameToCertificate()
//
//
//fmt.Println("Wait next call")
//text, _ = reader.ReadString('\n')
//fmt.Println(text)
  
//	proxy := core.NewMultipleHostReverseProxy(&url.URL{
//		Scheme: "http",
//		Host:   "localhost:8086",
//	})
//	cfg := &tls.Config{}
//
//	cert, err := tls.LoadX509KeyPair("/etc/letsencrypt/live/popular-design.fr/fullchain.pem", 
//	                                  "/etc/letsencrypt/live/popular-design.fr/privkey.pem")
//	if err != nil {
//		log.Fatal(err)
//	}
//	cfg.Certificates = append(cfg.Certificates, cert)
//	
//
//	server := http.Server{
//		Addr:      "localhost:9090",
//		Handler:   proxy,
//		TLSConfig: cfg,
//	}
//
//
//
//        ln, err := net.Listen("tcp", server.Addr)
//        if err != nil {
//                log.Fatal("error")
//        }
//
//        tlsListener := tls.NewListener(tcpKeepAliveListener{ln.(*net.TCPListener)}, cfg)
//        go server.Serve(tlsListener)
//
//
//
//	//go server.ListenAndServeTLS("", "")
//  
//  reader := bufio.NewReader(os.Stdin)
//fmt.Print("Enter text: ")
//text, _ := reader.ReadString('\n')
//fmt.Println(text)
//
//cert, err = tls.LoadX509KeyPair("/etc/letsencrypt/live/acebedo.fr/fullchain.pem", 
//	                                  "/etc/letsencrypt/live/acebedo.fr/privkey.pem")
//	if err != nil {
//		log.Fatal(err)
//	}
//	cfg.Certificates = append(cfg.Certificates, cert)
//
//	// keep adding remaining certs to cfg.Certificates
//
//	cfg.BuildNameToCertificate()
//
//
//fmt.Println("Wait next call")
//text, _ = reader.ReadString('\n')
//fmt.Println(text)

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
