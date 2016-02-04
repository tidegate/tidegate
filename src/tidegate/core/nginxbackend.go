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

/**************************************************/
/**************************************************/
type NGINXDaemon struct {
	configPath string
	binPath    string
	pidPath    string
	cmd        *exec.Cmd
}

func NewNGINXDaemon(configPath string, binPath string) (res *NGINXDaemon, err error) {
	res = &NGINXDaemon{}
	res.binPath, _ = filepath.Abs(binPath)
	res.configPath, _ = filepath.Abs(configPath)
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

func (self *NGINXDaemon) Start() (err error) {
	RootLogger.Debug("Starting NGINX daemon...")
	self.cmd = exec.Command(self.binPath, "-c", self.configPath)
	go self.cmd.Run()
	if err == nil {
	  RootLogger.Debug("NGINX daemon successfuly started")
	} else {
		err = errors.New(fmt.Sprintf("Unable to start NGINX daemon: %s", err.Error()))
	}
	return
}

func (self *NGINXDaemon) sendSignal(signal syscall.Signal) (err error) {
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
		RootLogger.Warningf("Unable to read PID file (%s). Are you sure NGINX is running ?", pidFilePath)
	}
	return
}
func (self *NGINXDaemon) Stop() (err error) {
	RootLogger.Debugf("Stopping nginx daemon")
	return self.sendSignal(syscall.SIGQUIT)
}

func (self *NGINXDaemon) Join() (err error) {
  err = self.cmd.Wait()
  return
}
func (self *NGINXDaemon) Status() bool {
	pidFilePath := filepath.Join(self.configPath, "nginx.pid")
	_, err := os.Open(pidFilePath)
	return os.IsExist(err)
}

func (self *NGINXDaemon) Reload() error {
	RootLogger.Debugf("Reloading nginx daemon")
	return self.sendSignal(syscall.SIGHUP)
}

/**************************************************/
/**************************************************/
type NGINXBackend struct {
	daemon         *NGINXDaemon
	configDirPath  string
	serversDirPath string
	tmpDirPath     string
	serverTemplate string
}

func NewNGINXBackend(runDirPath string, binDirPath string) (res *NGINXBackend, err error) {
	res = &NGINXBackend{}
	absOutputDirPath, err := filepath.Abs(runDirPath)
	if err != nil {
		err = errors.New(fmt.Sprintf("Invalid output directory path '%s': Unable to compute absolute path", runDirPath))
		return
	}

	res.configDirPath = filepath.Join(absOutputDirPath, "config")
	err = os.MkdirAll(res.configDirPath, 0700)
	if err == nil {
		RootLogger.Debugf("Successfuly created configuration directory '%s'", res.configDirPath)
	} else {
		err = errors.New(fmt.Sprintf("Unable to create configuration directory '%s'", res.configDirPath))
		return
	}

	res.tmpDirPath = filepath.Join(absOutputDirPath, "tmp")
	err = os.MkdirAll(res.tmpDirPath, 0700)
	if err == nil {
		RootLogger.Debugf("Successfuly created tmp directory '%s'", res.tmpDirPath)
	} else {
		err = errors.New(fmt.Sprintf("Unable to create tmp directory '%s'", res.tmpDirPath))
		return
	}

	logDirPath := filepath.Join(absOutputDirPath, "logs")
	err = os.MkdirAll(logDirPath, 0700)
	if err == nil {
		RootLogger.Debugf("Created NGINX logs directory '%s'", logDirPath)
	} else {
		err = errors.New(fmt.Sprintf("Unable to create NGINX logs directory '%s': %s", logDirPath, err.Error()))
		return
	}

	letsEncryptDirPath := filepath.Join(res.tmpDirPath, "letsencrypt")
	err = os.MkdirAll(letsEncryptDirPath, 0700)
	if err == nil {
		RootLogger.Debugf("Created LetsEncrypt directory '%s'", letsEncryptDirPath)
	} else {
		err = errors.New(fmt.Sprintf("Unable to create LetsEncrypt directory '%s': %s", letsEncryptDirPath, err.Error()))
		return
	}

	res.serversDirPath = filepath.Join(res.configDirPath, "servers")
	err = os.MkdirAll(res.serversDirPath, 0700)
	if err == nil {
		RootLogger.Debugf("Created servers directory '%s'", res.serversDirPath)
	} else {
		err = errors.New(fmt.Sprintf("Unable to create servers directory '%s': %s", res.serversDirPath, err.Error()))
		return
	}

	nginxConfFilePath := filepath.Join(res.configDirPath, "nginx.conf")
	fi, err := os.Create(nginxConfFilePath)
	if err != nil {
		err = errors.New(fmt.Sprintf("Unable to create NGINX configuration file '%s': %s", nginxConfFilePath, err.Error()))
		return
	}

	configTemplate := `
worker_processes 4;
pid {{.PidPath}};
error_log stderr;
daemon off;
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
     root {{.LetsEncryptDirPath}};
   } 
  }
  server {
     listen 443;
     charset utf-8;
     location "/.well-known" {
       root {{.LetsEncryptDirPath}};
     } 
  }
}`
	var t = template.New("NGINX Configuration")
	t, _ = t.Parse(configTemplate)
	err = t.Execute(fi, struct {
		PidPath                 string
		AccessLogPath           string
		ErrorLogPath            string
		ServerConfigurationPath string
		LetsEncryptDirPath      string
	}{
		filepath.Join(res.tmpDirPath, "nginx.pid"),
		filepath.Join(logDirPath, "http_access.log"),
		filepath.Join(logDirPath, "http_error.log"),
		filepath.Join(res.configDirPath, "servers", "*"),
		letsEncryptDirPath})
	fi.Close()
	if err == nil {
		RootLogger.Debugf("NGINX configuration file '%s' successfully generated", nginxConfFilePath)
	} else {
		err = errors.New(fmt.Sprintf("Unable to generate NGINX configuration file '%s': %s", nginxConfFilePath, err.Error()))
		return
	}
	res.daemon, err = NewNGINXDaemon(filepath.Join(res.configDirPath, "nginx.conf"), filepath.Join(binDirPath, "nginx"))
	if err == nil {
		RootLogger.Debugf("Docker daemon successfully created '%s'")
	} else {
		err = errors.New(fmt.Sprintf("Unable create NGINX daemon: %s", err.Error()))
		return
	}
	
	res.serverTemplate = `upstream {{ Replace .Domain "." "_" -1 }} {
  least_conn;
{{range .Endpoints}}  server {{.IP}}:{{.Port}} max_fails=3 fail_timeout=60 weight=1; 
{{end}}
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

}`

//{{if .IsSSL()}}
//    ssl on;
//    ssl_certificate      /etc/letsencrypt/live/$server.domain/fullchain.pem;
//    ssl_certificate_key  /etc/letsencrypt/live/$server.domain/privkey.pem;
//    ssl_session_cache  builtin:1000  shared:SSL:10m;
//    ssl_protocols  TLSv1 TLSv1.1 TLSv1.2;
//    ssl_ciphers HIGH:!aNULL:!eNULL:!EXPORT:!CAMELLIA:!DES:!MD5:!PSK:!RC4;
//    ssl_prefer_server_ciphers on;
//{{end}}
	return
}

func (self *NGINXBackend) Start() (err error) {
	go self.daemon.Start()
	return
}

func (self *NGINXBackend) Stop() (err error) {
	err = self.daemon.Stop()
	return
}

func (self *NGINXBackend) HandleEndpointCreation(server *Server, endpointId string) (err error) {
	RootLogger.Debugf("Generate configuration for '%s' %v", server.GetID(), len(server.Endpoints))
	funcMap := template.FuncMap{
		"Replace": strings.Replace,
	}
	t := template.New(fmt.Sprintf("Configuration for server %s", server.Domain)).Funcs(funcMap)
	t, err = t.Parse(self.serverTemplate)
	if err == nil {
		serverConfigFilePath := filepath.Join(self.serversDirPath, server.Domain)
		fi, err := os.Create(serverConfigFilePath)
		if err == nil {
			err = t.Execute(fi, *server)
			if(err != nil) {
			  err = errors.New(fmt.Sprintf("Unable to create server configuration file '%s': %s", serverConfigFilePath, err.Error()))
			  RootLogger.Errorf("Unable to create server configuration file '%s': %s", serverConfigFilePath, err.Error())
			  os.Remove(filepath.Join(self.serversDirPath, server.Domain))
			} else {
			  fi.Close()
			}
			
		} else {
			err = errors.New(fmt.Sprintf("Unable to create server configuration file '%s': %s", serverConfigFilePath, err.Error()))
		}
	} else {
		err = errors.New(fmt.Sprintf("Unable to generate configuration for server '%s': %s", server.Domain, err.Error()))
	}
	self.daemon.Reload()
	return
}
func (self *NGINXBackend) HandleEndpointDeletion(server *Server, endpointId string) (err error) {
	if len(server.Endpoints) == 0 {
		err = os.Remove(filepath.Join(self.serversDirPath, server.Domain))
	}
	self.daemon.Reload()
	return
}
