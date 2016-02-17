package core

import (
	"gopkg.in/alecthomas/kingpin.v2"
)

type TideGateConfiguration struct {
  DockerDaemonAddr string
  Verbose bool
  Quiet bool
  Syslog bool
} 

func ParseArgs(rawArgs []string) *TideGateConfiguration {
	var app = kingpin.New("tidegate", "Reverse proxy generator.")
	var verbose    = app.Flag("verbose", "Verbose mode.").Short('v').Bool()
	var quiet = app.Flag("quiet", "Quiet mode.").Short('q').Bool()
	var syslog = app.Flag("syslog", "Syslog backend.").Short('s').Bool()
	var dockerDaemonAddr = app.Flag("daemon-address", "Docker daemon address.").Short('d').Required().String()
	
	kingpin.MustParse(app.Parse(rawArgs))
	return &TideGateConfiguration{*dockerDaemonAddr, *verbose, *quiet, *syslog}
}
