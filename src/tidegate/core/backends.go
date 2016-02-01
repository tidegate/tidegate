package core

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"text/template"
)


type ReverseProxyConfigurationGenerator interface {
	GenerateConfigurations(servers *ServerStorage)
	GenerateConfiguration(server *Server)
}

type Backend interface {
	HandleServerCreation(server* Server)
	HandleServerDestruction(server* Server)
}

type NGINXRPConfigurationGenerator struct {
	outputDirPath string
	tmpDirPath string
}

func NewNGINXRPConfigurationGenerator(configDirPath string, tmpDirPath string) (res *NGINXRPConfigurationGenerator) {
	res = &NGINXRPConfigurationGenerator{}
	res.outputDirPath = configDirPath
	res.tmpDirPath = tmpDirPath
	return
}

type NGINXDaemon struct {
	configPath string
	binPath    string
	pidPath    string
	cmd        *exec.Cmd
}

func NewNGINXDaemon(configPath string, binPath string) (res *NGINXDaemon, err error) {
	res = &NGINXDaemon{}
	res.binPath = binPath
	res.configPath = configPath
	configContent, err := ioutil.ReadFile(configPath)
	if err == nil {
		lines := strings.Split(string(configContent), "\n")
	FileAnalysisLoop:
		for _, line := range lines {
			matched, _ := regexp.MatchString("pid [^;]*", line)
			if matched {
				re := regexp.MustCompile("pid (?P<Pidfile>[^;]*)")
				matches := re.FindStringSubmatch(line)
				if len(matches) > 0 {
					RootLogger.Debugf("PID file for nginx daemon is '%v'", matches[1])
					res.pidPath = matches[1]
				} else {
					err = errors.New(fmt.Sprintf("Invalid 'pid' directive the config '%s'", configPath))
				}
				break FileAnalysisLoop
			}
		}
	} else {
		err = errors.New(fmt.Sprintf("Unable to find 'pid' directive in '%s'", configPath))
	}
	return
}

type NGINXBackend struct {
	daemon          NGINXDaemon
	configGenerator NGINXRPConfigurationGenerator
}

func NewNGINXBackend(runDirPath string, binDirPath string) (res *NGINXBackend, err error) {
	var configDirPath = filepath.Join(runDirPath, "config")
	err = os.MkdirAll(configDirPath, 0700)
	
	var tmpDirPath = filepath.Join(runDirPath, "tmp")
	err = os.MkdirAll(tmpDirPath, 0700)
	
	res = &NGINXBackend{}
	daemon, err := NewNGINXDaemon(filepath.Join(configDirPath, "nginx.conf"), filepath.Join(binDirPath,"nginx"))
	if err == nil {
		res.daemon = *daemon
		res.configGenerator = *NewNGINXRPConfigurationGenerator(configDirPath,tmpDirPath)
	}
	return
}

func (self NGINXBackend) Start() (err error) {
	err = self.daemon.Start()
	return
}

func (self NGINXBackend) 	HandleServerCreation() {
  self.daemon.Reload()
}

func (self NGINXBackend) HandleServerDestruction() {
  self.daemon.Reload()
}

func (self NGINXRPConfigurationGenerator) GenerateConfigurations(servers *ServerStorage) {
	configTemplate := `
worker_processes 4;
pid {{.PidPath}};
error_log stderr;

events {
	worker_connections 768;
	# multi_accept on;
}

http {
	##
	# Basic Settings
	##
	sendfile on;
	tcp_nopush on;
	tcp_nodelay on;
	keepalive_timeout 65;
	types_hash_max_size 2048;
	             
	include /etc/nginx/mime.types;
	default_type application/octet-stream;

	##
	# SSL Settings
	##
	ssl_protocols TLSv1 TLSv1.1 TLSv1.2; # Dropping SSLv3, ref: POODLE
	ssl_prefer_server_ciphers on;

	##
	# Logging Settings
	##
	access_log {{.AccessLogPath}};
	error_log {{.ErrorLogPath}};

	##
	# Gzip Settings
	##
	gzip on;
	gzip_disable "msie6";

	##
	# Virtual Host Configs
	##
	include {{.ServerConfigurationPath}};
	
	server {
   listen 80;
   charset utf-8;
   location "/.well-known" {
     root /tmp/letsencrypt;
   } 
  }
  server {
     listen 443;
     charset utf-8;
     location "/.well-known" {
       root /tmp/letsencrypt;
     } 
  }
}`

	var t = template.New("NGINX Configuration")
	t, _ = t.Parse(configTemplate)
	err := os.MkdirAll(self.outputDirPath, 0700)
	err = os.MkdirAll(filepath.Join(self.outputDirPath, "logs"), 0700)
	fi, err := os.Create(filepath.Join(self.outputDirPath, "nginx.conf"))
	if err != nil {
		RootLogger.Fatalf(err.Error())
	}
	err = t.Execute(fi, struct {
		PidPath                 string
		AccessLogPath           string
		ErrorLogPath            string
		ServerConfigurationPath string
	}{
		filepath.Join(self.outputDirPath, "nginx.pid"),
		filepath.Join(self.outputDirPath, "logs", "http_access.log"),
		filepath.Join(self.outputDirPath, "logs", "http_error.log"),
		filepath.Join(self.outputDirPath, "servers", "*")})
	if err != nil {
		fmt.Println("There was an error:", err.Error())
	}

	for _, server := range *servers.GetServers() {
		self.GenerateConfiguration(&server)
	}
}

func (self NGINXRPConfigurationGenerator) GenerateConfiguration(server *Server) {
	var funcMap = template.FuncMap{
		"Replace": strings.Replace,
	}

	var serverTemplate string = `upstream {{ Replace .Domain "." "_" -1 }} {
  least_conn;
{{range .Endpoints}}  server {{.IP}}:{{.Port}} max_fails=3 fail_timeout=60 weight=1; {{end}}
}

server {
   listen {{.ExternalPort}};
   server_name {{.Domain}};
   charset utf-8;
 
   location / {
     proxy_pass http://{{ Replace .Domain "." "_" -1 }}/;
     proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
     proxy_set_header Host $host;
     proxy_set_header X-Real-IP $remote_addr;
     proxy_set_header X-Forwarded-Proto $scheme;
   }
{{if .SSLEnabled}}
    ssl on;
    ssl_certificate      /etc/letsencrypt/live/$server.domain/fullchain.pem;
    ssl_certificate_key  /etc/letsencrypt/live/$server.domain/privkey.pem;
    ssl_session_cache  builtin:1000  shared:SSL:10m;
    ssl_protocols  TLSv1 TLSv1.1 TLSv1.2;
    ssl_ciphers HIGH:!aNULL:!eNULL:!EXPORT:!CAMELLIA:!DES:!MD5:!PSK:!RC4;
    ssl_prefer_server_ciphers on;
{{end}}
}`

	var t = template.New(fmt.Sprintf("Configuration for server %s", server.Domain)).Funcs(funcMap)
	t, _ = t.Parse(serverTemplate)
	var serverPath = filepath.Join(self.outputDirPath, "servers")
	var err = os.MkdirAll(serverPath, 0700)
	fi, err := os.Create(filepath.Join(serverPath, server.Domain))
	if err != nil {
		RootLogger.Fatalf(err.Error())
	}
	err = t.Execute(fi, *server)
	if err != nil {
		fmt.Println("There was an error:", err.Error())
	}
}
func (self NGINXDaemon) Start() (err error) {
	self.cmd = exec.Command(self.binPath,"-c",self.configPath)
	err = self.cmd.Start()
	RootLogger.Debugf("%s %s %s", self.binPath,"-c", self.configPath)
	
	if err == nil {
	  err = self.cmd.Wait()
	  if err == nil {
		  RootLogger.Debug("Nginx successfuly started")
		} else {
		  err = errors.New(fmt.Sprintf("Unable to start the daemon: %s", err.Error()))
		}
	} else {
	  err = errors.New(fmt.Sprintf("Unable to start the daemon: %s", err.Error()))
	}
	
	return
}

func (self NGINXDaemon) sendSignal(signal syscall.Signal) (err error) {
	pidFilePath := filepath.Join(self.configPath, "nginx.pid")
	dat, err := ioutil.ReadFile(pidFilePath)
	if err == nil {
		pid, err := strconv.Atoi(string(dat[0 : len(dat)-1]))
		if err != nil {
			RootLogger.Debugf("Sending signal '%v' to nginx (pid:(%v))", signal, pid)
			err = syscall.Kill(pid, signal)
			if err != nil {
				RootLogger.Warningf("Unable send signal (%s).", err.Error())
			}
		} else {
			RootLogger.Warningf("Invalid value in PID file (%s).", pidFilePath)
		}
	} else {
		RootLogger.Warningf("Unable to read PID file (%s).", pidFilePath)
	}
	return
}
func (self NGINXDaemon) Stop() (err error) {
  RootLogger.Debugf("Stopping nginx daemon")
	return self.sendSignal(syscall.SIGQUIT)
}

func (self NGINXDaemon) Status() bool {
	pidFilePath := filepath.Join(self.configPath, "nginx.pid")
	_, err := os.Open(pidFilePath)
	return os.IsExist(err)
}

func (self NGINXDaemon) Reload() error {
  RootLogger.Debugf("Reloading nginx daemon")
	return self.sendSignal(syscall.SIGHUP)
}
