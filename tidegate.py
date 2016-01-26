#!/usr/bin/env python3

import argparse
import logging
import requests
from docker import Client
import sys
import threading

ROOTLOGGER = logging.getLogger("tidegate")

class DockerEventMonitorThread(threading.Thread):

    def __init__(self, docker_addr):
        threading.Thread.__init__(self)
        self.running = False
        self.docker_addr = docker_addr
        
    def run(self):
        cli = Client(base_url=self.docker_addr)
        events = cli.events()
        self.running = True
        while self.running:
          for e in events :
            print(e)

    def stop(self):
        ROOTLOGGER.info("CFDNSUpdater is stopping")
        self.running = False
  
class TideGate:

    def init_logs(type):
        formatter = logging.Formatter('%(message)s')
        hdlr = None
        if type == True:
          hdlr = SysLogHandler(address='/dev/log',
          facility=SysLogHandler.LOG_DAEMON)
        else:
          hdlr = StreamHandler(sys.stdout)
          hdlr.setLevel(1)
          hdlr.setFormatter(formatter)
          ROOTLOGGER.addHandler(hdlr)

    @staticmethod
    def parse_arguments(raw_args):
        parser = argparse.ArgumentParser(prog="TideGate",
                             description='Automatic reverse proxy.')

        parser.add_argument('--docker',
                             required=True,
                             help='Docker socket address',
                             type=str)

        return parser.parse_args(raw_args)

    @staticmethod
    def main(raw_args):
        args = TideGate.parse_arguments(raw_args)
        thread = DockerEventMonitorThread(args.docker)
        thread.start()
        thread.join()

if __name__ == "__main__":
    sys.exit(TideGate.main(sys.argv[1:]))

