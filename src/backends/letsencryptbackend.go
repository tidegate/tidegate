package backends

import (
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"time"
	"github.com/aacebedo/tidegate/src/core"
	"github.com/aacebedo/tidegate/src/patterns"
	"github.com/aacebedo/tidegate/src/servers"
)

type CertificateUpdateEvent struct {
	Domains []string
}

/**************************************************/
/**************************************************/
type LetsEncryptBackend struct {
	//Observer patterns.Observer
}

func NewLetsEncryptBackend() (res *LetsEncryptBackend) {
	res = &LetsEncryptBackend{}
	//res.Observer = patterns.NewBasicObserver(res)
	return
}

//func (self *LetsEncryptBackend) HandleEvent(value interface{}) {
//	self.Observer.Update(value)
//}

func (self *LetsEncryptBackend) HandleEvent(value interface{}) {
	switch value.(type) {
	case *core.ServerAdditionEvent:
		event := value.(*core.ServerAdditionEvent)
		logger.Debugf("Handling server '%s' creation", event.Server.GetId())
		self.HandleServerAddition(event.Server)
	case *core.ServerRemovalEvent:
		event := value.(*core.ServerRemovalEvent)
		logger.Debugf("Handling server '%s' removal", event.Server.GetId())
		self.HandleServerRemoval(event.Server)
	default:
		logger.Debugf("Event of type '%s' ignored", reflect.TypeOf(value))
	}
}

func (self *LetsEncryptBackend) HandleServerAddition(server *servers.Server) (err error) {
	rootDomain, _ := server.GetRootDomain()
	fileContent, rerr := ioutil.ReadFile(filepath.Join("certs", rootDomain))
	if rerr != nil {
		err = errors.New(fmt.Sprintf("Unable to read configuration file '%s'", rerr.Error()))
		logger.Debugf("Unable to read configuration file '%s'", rerr.Error())
		return
	}
	certs, err := x509.ParseCertificates([]byte(fileContent))

	if err != nil {
		err = errors.New(fmt.Sprintf("Unable to read certificate '%s'", err.Error()))
		logger.Debugf("Unable to read certificate  '%s'", err.Error())
		return
	}

	if len(certs) == 1 {
		cert := certs[0]
		logger.Debugf("%v", cert.DNSNames)
		found := false
	DOMAIN_SEARCH:
		for _, domain := range cert.DNSNames {
			if server.Domain == domain {
				found = true
				break DOMAIN_SEARCH
			}
		}
		if found {
			logger.Debugf("Subdomain '%s' is already in certificate", server.Domain)
		} else {
			logger.Debugf("Subdomain '%s' is not certificate, certificate needs to be generated", server.Domain)
		}
	}

	//err = yaml.Unmarshal([]byte(fileContent), &res.DomainConfigs)
	logger.Debugf("Certificate for server '%s' has been generated", rootDomain)
	return
}

func (self *LetsEncryptBackend) HandleServerRemoval(server *servers.Server) (err error) {
	logger.Debugf("Certificate for server '%s' has been removed", server.Domain)
	return
}

type CertificateMonitor struct {
	CertFilePath string
	observer     patterns.BasicObserver
}

func NewCertificateMonitor(certFilePath string) (res *CertificateMonitor) {
	res = &CertificateMonitor{}
	res.CertFilePath = certFilePath
	res.observer = *patterns.NewBasicObserver(res)
	return
}

func (self *CertificateMonitor) HandleUpdate(value interface{}) {
	switch value.(type) {
	case *CertificateUpdateEvent:
		fileContent, rerr := ioutil.ReadFile(self.CertFilePath)
		if rerr != nil {
			logger.Debugf("File '%s' was not found, generate it", self.CertFilePath)
			
		} else {
		  logger.Debugf("File '%s' found, parsing it to check its information", self.CertFilePath)
		  certs, err := x509.ParseCertificates([]byte(fileContent))
    	if err != nil {
    		logger.Warningf("Unable to parse file '%s': %s",self.CertFilePath, err.Error())
    		return
    	}
      if len(certs) < 1 {
  	    logger.Warningf("Unable to parse file '%s': No certificate found.",self.CertFilePath)
    		return
      }
    	if len(certs) >= 1 {
    		cert := certs[0]
    		logger.Debugf("Certificate in file '%s':",self.CertFilePath)
    		logger.Debugf("  Subject: %s",cert.Subject)
    		logger.Debugf("  Validity: Not Before '%v' and Not After '%v'",cert.NotBefore,cert.NotAfter)
    		logger.Debugf("  Alternative Names: '%v'",cert.DNSNames)
    		now := time.Now()
    		if now.Add(3600*time.Minute).Unix() > cert.NotAfter.Unix() {
    		  logger.Infof("Certificate  '%s' will not be valid in 24 hours, It needs to be regenerated",self.CertFilePath)
    		}
    		}
    	}
	default:
		logger.Debugf("Event of type '%s' ignored", reflect.TypeOf(value))
	}
}
