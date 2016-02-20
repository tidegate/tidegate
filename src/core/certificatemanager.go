package core

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xenolf/lego/acme"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
)

type CertificateGenerator interface {
	GenerateCertificate(subdomains []string, domain string, userMail string) (res *tls.Certificate, err error)
}

type CertificateLoader interface {
	Load() (res []*tls.Certificate, err error)
}

type user struct {
	email        string
	registration *acme.RegistrationResource
	key          *rsa.PrivateKey
}

func (u user) GetEmail() string {
	return u.email
}
func (u user) GetRegistration() *acme.RegistrationResource {
	return u.registration
}
func (u user) GetPrivateKey() *rsa.PrivateKey {
	return u.key
}

type LetsEncryptCertificateGenerator struct {
	outputDirPath string
	serverAddr    string
	httpPort      uint
	httpsPort     uint
	keySize       uint
}

func NewLetsEncryptCertificateGenerator(outputDirPath string,
	serverAddr string,
	httpPort uint,
	httpsPort uint,
	keySize uint) (res *LetsEncryptCertificateGenerator, err error) {
	stat, err := os.Stat(outputDirPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(outputDirPath, 0777)
			if err != nil {
				return
			}
		} else {
			return
		}
	} else {
		if !stat.IsDir() {
			err = errors.New(fmt.Sprintf("%v exists but is not a dir, change certificate output directory or remove the file", outputDirPath))
			return
		}
	}
	res = &LetsEncryptCertificateGenerator{outputDirPath,
		serverAddr,
		httpPort,
		httpsPort,
		keySize}
	return
}

func (self *LetsEncryptCertificateGenerator) GenerateCertificate(subdomains []string,
	domain string, userMail string) (res *tls.Certificate, err error) {
	IsFileOrNotExist := func(filePath string) (res bool) {
		stat, staterr := os.Stat(filePath)
		if staterr != nil {
			res = os.IsNotExist(staterr)
		} else {
			res = !stat.IsDir()
		}
		return
	}

	certOutDirPath := filepath.Join(self.outputDirPath, domain)
	err = os.MkdirAll(certOutDirPath, 0777)
	if err != nil {
		return
	}
	certOut := filepath.Join(certOutDirPath, "cert.pem")
	if !IsFileOrNotExist(certOut) {
		err = errors.New(fmt.Sprintf("'%v' exists and is not a file", certOut))
		return
	}

	privOut := filepath.Join(certOutDirPath, "privkey.pem")
	if !IsFileOrNotExist(privOut) {
		err = errors.New(fmt.Sprintf("'%v' exists and is not a file", privOut))
		return
	}

	metaOut := filepath.Join(certOutDirPath, "metadata.json")
	if !IsFileOrNotExist(metaOut) {
		err = errors.New(fmt.Sprintf("'%v' exists and is not a file", metaOut))
		return
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, int(self.keySize))
	if err != nil {
		return
	}
	user := user{
		email: userMail,
		key:   privateKey,
	}
	client, err := acme.NewClient(self.serverAddr, &user, int(self.keySize))
	if err != nil {
		return
	}
	client.SetHTTPAddress(fmt.Sprintf(":%v", self.httpPort))
	client.SetTLSAddress(fmt.Sprintf(":%v", self.httpsPort))

	user.registration, err = client.Register()
	if err != nil {
		return
	}

	err = client.AgreeToTOS()
	if err != nil {
		return
	}
	targetDomains := []string{}

	for _, subdomain := range subdomains {
		targetDomains = append(targetDomains, fmt.Sprintf("%v.%v", subdomain, domain))
	}

	certRes, failures := client.ObtainCertificate(targetDomains, false, nil)
	if len(failures) > 0 {
		err = errors.New(fmt.Sprintf("Unable to obtain certificate: %s", failures))
		return
	}

	cert, err := tls.X509KeyPair(certRes.Certificate, certRes.PrivateKey)
	if err != nil {
		return
	}

	err = ioutil.WriteFile(certOut, certRes.Certificate, 0600)
	if err != nil {
		return
	}
	err = ioutil.WriteFile(privOut, certRes.PrivateKey, 0600)
	if err != nil {
		return
	}
	jsonBytes, err := json.MarshalIndent(cert, "", "\t")
	if err != nil {
		return
	}
	err = ioutil.WriteFile(metaOut, jsonBytes, 0600)
	if err != nil {
		return
	}

	res = &cert
	return
}

type CertificateManager struct {
	generator    CertificateGenerator
	certificates map[string]map[string]*tls.Certificate
}

func NewCertificateManager(generator CertificateGenerator) (res *CertificateManager) {
	res = &CertificateManager{}
	res.generator = generator
	res.certificates = make(map[string]map[string]*tls.Certificate)
	return
}

type DomainName struct {
	Subdomain string
	Domain    string
	Ext       string
}

func (self *DomainName) GetRootDomain() (res string) {
	res = fmt.Sprintf("%v.%v", self.Domain, self.Ext)
	return
}
func ParseDomainName(domain string) (res *DomainName, err error) {
	re, err := regexp.Compile(`^(?P<subdomain>[a-z0-9][a-z0-9\-]{1,61}[a-z0-9])\.(?P<domain>[a-z0-9][a-z0-9\-]{1,61}[a-z0-9])\.(?P<ext>[a-z]{2,63})$`)
	if err != nil {
		return
	}
	groups := make(map[string]string)
	match := re.FindStringSubmatch(domain)

	if len(match) != 4 {
		err = errors.New("Invalid domain name")
		return
	}
	for i, name := range re.SubexpNames() {
		if i != 0 {
			groups[name] = match[i]
		}
	}
	subdomain, hasSubdomain := groups["subdomain"]
	domain, hasDomain := groups["domain"]
	ext, hasExt := groups["ext"]
	if !hasSubdomain || !hasDomain || !hasExt {
		err = errors.New(fmt.Sprintf("Unable to find subdomain, domain or extension in %v", domain))
	} else {
		res = &DomainName{subdomain, domain, ext}
	}
	return
}

func (self *CertificateManager) LoadCertificates(certLoader CertificateLoader) (res bool, err error) {
	certificates, err := certLoader.Load()
	if err != nil {
		return
	}

	for _, tlscert := range certificates {
		x509cert, parseErr := x509.ParseCertificates(tlscert.Certificate[0])
		if parseErr == nil {
			for _, dnsName := range x509cert[0].DNSNames {
				domainName, loadErr := ParseDomainName(dnsName)
				if loadErr == nil {
					self.certificates[domainName.GetRootDomain()] = make(map[string]*tls.Certificate)
					self.certificates[domainName.GetRootDomain()][domainName.Subdomain] = tlscert
					logger.Debugf("Attached certificate for '%v'", dnsName)
				} else {
					logger.Warningf("Unable to attach certificate for %v: %v", dnsName, loadErr)
				}
			}
		} else {
			logger.Warningf("Unable to parse certificate")
		}
	}
	return
}

func (self *CertificateManager) Contains(domain string) (res bool, err error) {
	domainName, err := ParseDomainName(domain)
	if err != nil {
		return
	}
	_, res = self.certificates[domainName.GetRootDomain()][domainName.Subdomain]
	return
}

func (self *CertificateManager) GetCertificate(domain string) (res *tls.Certificate, err error) {
	domainName, err := ParseDomainName(domain)
	if err != nil {
		return
	}
	var subdomains []string
	if domainMap, containsRootDomain := self.certificates[domainName.GetRootDomain()]; containsRootDomain {
		if cert, containsSubdomain := domainMap[domainName.Subdomain]; containsSubdomain {
			logger.Debugf("Certificate found for '%v'.", domain)
			res = cert
			return
		}
		for subdomain, _ := range domainMap {
			subdomains = append(subdomains, subdomain)
		}
	} else {
		self.certificates[domainName.GetRootDomain()] = make(map[string]*tls.Certificate)
	}
	subdomains = append(subdomains, domainName.Subdomain)
	logger.Debugf("No certificate found for '%v', generates it.", domain)
	cert, err := self.generator.GenerateCertificate(subdomains, domainName.GetRootDomain(), fmt.Sprintf("admin@%v", domainName.GetRootDomain()))
	for _, subdomain := range subdomains {
		self.certificates[domainName.GetRootDomain()][subdomain] = cert
	}
	return
}

type FileCertificateLoader struct {
	dirToLoadPath string
}

func NewFileCertificateLoader(dirToLoadPath string) (res *FileCertificateLoader) {
	res = &FileCertificateLoader{dirToLoadPath}
	return
}

func (self *FileCertificateLoader) Load() (res []*tls.Certificate, err error) {
	els, _ := ioutil.ReadDir(self.dirToLoadPath)
	for _, file := range els {
		if file.IsDir() {
			dirPath := filepath.Join(self.dirToLoadPath, file.Name())
			cert, loadErr := tls.LoadX509KeyPair(filepath.Join(dirPath, "cert.pem"),
				filepath.Join(dirPath, "privkey.pem"))
			if loadErr == nil {
				logger.Debugf("Certificate successfully loaded from '%v'", dirPath)
				res = append(res, &cert)
			} else {
				logger.Warningf("Unable to load certificate from '%v': %V", dirPath, loadErr)
			}
		}
	}
	return
}
