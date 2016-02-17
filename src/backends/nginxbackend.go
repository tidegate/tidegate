package backends

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"text/template"
	"github.com/aacebedo/tidegate/src/servers"
)

/**************************************************/
/**************************************************/
type NGINXDaemon struct {
	configPath  string
	binPath     string
	pidFilePath string
	cmd         *exec.Cmd
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
					logger.Debugf("PID file for nginx daemon is '%v'", matches[1])
					res.pidFilePath = matches[1]
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
	logger.Debug("Starting NGINX daemon...")
	self.cmd = exec.Command(self.binPath, "-c", self.configPath)
	go self.cmd.Run()
	if err == nil {
		logger.Debug("NGINX daemon successfuly started")
	} else {
		err = errors.New(fmt.Sprintf("Unable to start NGINX daemon: %s", err.Error()))
	}
	return
}

func (self *NGINXDaemon) sendSignal(signal syscall.Signal) (err error) {
	dat, err := ioutil.ReadFile(self.pidFilePath)
	if err == nil {
		pid, err := strconv.Atoi(string(dat[0 : len(dat)-1]))
		if err == nil {
			logger.Debugf("Sending signal '%v' to nginx (pid:(%v))", signal, pid)
			err = syscall.Kill(pid, signal)
			if err != nil {
				logger.Warningf("Unable send signal (%s).", err.Error())
			}
		} else {
			logger.Warningf("Invalid value in PID file (%s).", self.pidFilePath)
		}
	} else {
		logger.Warningf("Unable to read PID file (%s). Are you sure NGINX is running ?", self.pidFilePath)
	}
	return
}
func (self *NGINXDaemon) Stop() (err error) {
	logger.Debugf("Stopping nginx daemon")
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
	logger.Debugf("Reloading nginx daemon")
	return self.sendSignal(syscall.SIGHUP)
}

/**************************************************/
/**************************************************/
type NGINXBackend struct {
	daemon         *NGINXDaemon
	configDirPath  string
	serversDirPath string
	tmpDirPath     string
	certsDirPath     string
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
		logger.Debugf("Successfuly created configuration directory '%s'", res.configDirPath)
	} else {
		err = errors.New(fmt.Sprintf("Unable to create configuration directory '%s'", res.configDirPath))
		return
	}

	res.tmpDirPath = filepath.Join(absOutputDirPath, "tmp")
	err = os.MkdirAll(res.tmpDirPath, 0700)
	if err == nil {
		logger.Debugf("Successfuly created tmp directory '%s'", res.tmpDirPath)
	} else {
		err = errors.New(fmt.Sprintf("Unable to create tmp directory '%s'", res.tmpDirPath))
		return
	}

	logDirPath := filepath.Join(absOutputDirPath, "logs")
	err = os.MkdirAll(logDirPath, 0700)
	if err == nil {
		logger.Debugf("Created NGINX logs directory '%s'", logDirPath)
	} else {
		err = errors.New(fmt.Sprintf("Unable to create NGINX logs directory '%s': %s", logDirPath, err.Error()))
		return
	}
	
	res.certsDirPath = filepath.Join(absOutputDirPath, "certs")
	err = os.MkdirAll(res.certsDirPath, 0700)
	if err == nil {
		logger.Debugf("Created NGINX certificates directory '%s'", res.certsDirPath)
	} else {
		err = errors.New(fmt.Sprintf("Unable to create NGINX certs directory '%s': %s", res.certsDirPath, err.Error()))
		return
	}
	
	res.serversDirPath = filepath.Join(res.configDirPath, "servers")
	err = os.MkdirAll(res.serversDirPath, 0700)
	if err == nil {
		logger.Debugf("Created servers directory '%s'", res.serversDirPath)
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
	# Logging Settings
	##
	access_log {{.AccessLogPath}};
	error_log {{.ErrorLogPath}};

	##
	# Gzip Settings
	##
	gzip on;
	gzip_disable "msie6";

 	server {
        return 404;
  }
	##
	# Virtual Host Configs
	##
	include {{.ServerConfigurationPath}};

}`
	var t = template.New("NGINX Configuration")
	t, _ = t.Parse(configTemplate)
	err = t.Execute(fi, struct {
		PidPath                 string
		AccessLogPath           string
		ErrorLogPath            string
		ServerConfigurationPath string
	}{
		filepath.Join(res.tmpDirPath, "nginx.pid"),
		filepath.Join(logDirPath, "http_access.log"),
		filepath.Join(logDirPath, "http_error.log"),
		filepath.Join(res.configDirPath, "servers", "*")})
	fi.Close()
	if err == nil {
		logger.Debugf("NGINX configuration file '%s' successfully generated", nginxConfFilePath)
	} else {
		err = errors.New(fmt.Sprintf("Unable to generate NGINX configuration file '%s': %s", nginxConfFilePath, err.Error()))
		return
	}
	res.daemon, err = NewNGINXDaemon(filepath.Join(res.configDirPath, "nginx.conf"), filepath.Join(binDirPath, "nginx"))
	if err == nil {
		logger.Debugf("Docker daemon successfully created '%s'")
	} else {
		err = errors.New(fmt.Sprintf("Unable create NGINX daemon: %s", err.Error()))
		return
	}

	go res.daemon.Start()
	return
}

func generateUpstreamName(serverId servers.ServerId) (res string) {
  res = strings.Replace(string(serverId),".","_",-1)
  return
}
func (self *NGINXBackend) HandleEndpointAddition(server *servers.Server) (err error) {
	logger.Debugf("Generate configuration for '%s' %v", server.GetId(), len(server.Endpoints))

  serverTemplate := `upstream {{ generateUpstreamName .Server.GetId  }} {
least_conn;
{{range .Server.Endpoints}}  server {{.IP}}:{{.Port}} max_fails=3 fail_timeout=60 weight=1; 
{{end}}
}

server {

  if ($host != {{.Server.Domain}}) {
        return 403;
   }
   charset utf-8;
   server_name {{.Server.Domain}};
   {{if .Server.IsSSL}}
     listen {{.Server.ExternalPort}} ssl;
      ssl_certificate      {{ .CertsDirPath }}/{{.Server.GetRootDomain }}/fullchain.pem;
      ssl_certificate_key  {{ .CertsDirPath }}/{{.Server.GetRootDomain }}/privkey.pem;
      ssl_protocols  TLSv1 TLSv1.1 TLSv1.2;
      ssl_ciphers HIGH:!aNULL:!eNULL:!EXPORT:!CAMELLIA:!DES:!MD5:!PSK:!RC4;
      ssl_prefer_server_ciphers on;
      location / {
        proxy_pass http://{{ generateUpstreamName .Server.GetId }}/;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_redirect http://{{generateUpstreamName .Server.GetId}} https://{{.Server.Domain}};
      }
  {{else}}
     listen {{.Server.ExternalPort}};
     location / {
       proxy_pass http://{{ generateUpstreamName .Server.GetId }}/;
       proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
       proxy_set_header Host $host;
       proxy_set_header X-Real-IP $remote_addr;
       proxy_set_header X-Forwarded-Proto $scheme;
     }  
  {{end}}
}

#server {
#   charset utf-8;
#   server_name {{.Server.Domain}};
#   location "/.well-known" {
#   root {{ .TmpDirPath }};
#   }
# }`

	if len(server.Endpoints) > 0 {
		funcMap := template.FuncMap{
			"generateUpstreamName": func(serverId servers.ServerId) string {return strings.Replace(string(serverId),".","_",-1)},
		}
		t := template.New(fmt.Sprintf("Configuration for server %s", server.Domain)).Funcs(funcMap)
		t, err = t.Parse(serverTemplate)
		if err == nil {
			serverConfigFilePath := filepath.Join(self.serversDirPath, string(server.GetId()))
			fi, err := os.Create(serverConfigFilePath)
			if err == nil {
				err = t.Execute(fi, struct {
		                        Server   *servers.Server
                        		CertsDirPath      string
                        		TmpDirPath      string
                        	}{server,self.certsDirPath,self.tmpDirPath})
				if err != nil {
					err = errors.New(fmt.Sprintf("Unable to create server configuration file '%s': %s", serverConfigFilePath, err.Error()))
					logger.Errorf("Unable to create server configuration file '%s': %s", serverConfigFilePath, err.Error())
					os.Remove(filepath.Join(self.serversDirPath, server.Domain))
				} else {
					fi.Sync()
					fi.Close()
				}
			} else {
				err = errors.New(fmt.Sprintf("Unable to create server configuration file '%s': %s", serverConfigFilePath, err.Error()))
			}
		} else {
			err = errors.New(fmt.Sprintf("Unable to generate configuration for server '%s': %s", server.Domain, err.Error()))
		}
	} else {
		os.Remove(filepath.Join(self.serversDirPath, server.Domain))
	}

	self.daemon.Reload()
	return
}
func (self *NGINXBackend) HandleEndpointRemoval(server *servers.Server) (err error) {
	if len(server.Endpoints) == 0 {
		err = os.Remove(filepath.Join(self.serversDirPath, string(server.GetId())))
	}
	self.daemon.Reload()
	return
}

func (self *NGINXBackend) HandleEvent(value interface{}) {
	switch value.(type) {
	case *servers.EndpointAdditionEvent:
		event := value.(*servers.EndpointAdditionEvent)
		self.HandleEndpointAddition(event.Server)
	case *servers.EndpointRemovalEvent:
		event := value.(*servers.EndpointRemovalEvent)
		self.HandleEndpointRemoval(event.Server)
	default:
		logger.Warningf("Event of type '%s' ignored", reflect.TypeOf(value))
	}
}
