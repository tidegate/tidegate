package core

import (
	"github.com/op/go-logging"
	"os"
)

var logger = logging.MustGetLogger("tidegate")

func InitLoggers(verbose bool, quiet bool, syslog bool) {
	format := logging.MustStringFormatter(`%{color}%{time:15:04:05.000} | %{longfunc}  %{level:.10s} ▶%{color:reset} %{message}`)
	var outstream * os.File
	if quiet {
		outstream = os.NewFile(uintptr(3), "/dev/null")
	} else {
		outstream = os.Stderr
	}

	backend := logging.NewLogBackend(outstream, "", 0)
	formatter := logging.NewBackendFormatter(backend, format)
	leveledBackend := logging.AddModuleLevel(formatter)

	if verbose {
		leveledBackend.SetLevel(logging.DEBUG, "")
	} else {
		leveledBackend.SetLevel(logging.INFO, "")
	}

	logging.SetBackend(leveledBackend)

}
