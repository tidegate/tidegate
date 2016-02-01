#!/usr/bin/env python3
import traceback
import argparse
import logging
from docker import Client
import sys
import threading
from enum import Enum
import json
from logging import StreamHandler
from colorlog import ColoredFormatter
import core
from datetime import datetime
from backends import NginxReverseProxyConfigGenerator

  
ROOTLOGGER = logging.getLogger("tidegate")

class DockerEventMonitorThread(threading.Thread):
    def __init__(self,processor,client,start_time):
        threading.Thread.__init__(self)
        self.running = False
        self.processor = processor
        self.client = client
        self.start_time = start_time
        
    def run(self):
        events = self.client.events(since=self.start_time,decode=True)
        self.running = True 
        while self.running:
          for e in events :
            self.processor.process_event(e)
            
    def stop(self):
        ROOTLOGGER.info("Tidegate is stopping")
        self.running = False
        
class DockerEventProcessor:
  def __init__(self,client,storage,loader):
    self.services = set()
    self.client = client
    self.processing_functions = {"create":DockerEventProcessor.process_create_event, 
                                 "die":DockerEventProcessor.process_die_event,
                                 "start":DockerEventProcessor.process_start_event, 
                                 "destroy":DockerEventProcessor.process_destroy_event}
    self.storage = storage
    self.loader = loader

  def process_event(self,event):
    print(event)
    if "status" in event and "id" in event:
      if event["status"] in self.processing_functions:
        self.processing_functions[event["status"]](self,event["id"])
      else:
        ROOTLOGGER.debug("Event '{}' of container '{}' ignored: unhandled status".format(event["status"],event["id"]))
    else:
      if "id" in event:
        ROOTLOGGER.warning("Event of container '{}' ignored: missing status".format(event["id"]))
      else:
        ROOTLOGGER.warning("Event ignored: missing id")
    
  def process_create_event(self,container_id):
    #self.services.add(container_id)
    ROOTLOGGER.debug("Container '{}' created".format(container_id))
    
  def process_destroy_event(self,container_id):
    #self.services.remove(container_id)
    ROOTLOGGER.debug("Container '{}' destroyed".format(container_id))
  
  def process_die_event(self,container_id):
    ROOTLOGGER.debug("Container '{}' died".format(container_id))
  
  def process_start_event(self, container_id):
    server = self.loader.load_server(container_id)
    if server != None:
      self.storage.add(server)
            #traceback.print_exc(e)
 
    ROOTLOGGER.info("Container '{}' started".format(container_id))
  

  
class TideGate:
    @staticmethod
    def init_logs():
#         formatter = ColoredFormatter(
#             "%(log_color)s[%(levelname)-8s]%(reset)s %(message)s",
#             datefmt=None,
#             reset=True,
#             log_colors={
#                 'DEBUG': 'cyan',
#                 'INFO': 'green',
#                 'WARNING': 'yellow',
#                 'ERROR': 'red',
#                 'CRITICAL': 'bold_red'
#             })
        formatter = logging.Formatter('[%(levelname)-8s] %(message)s')
        hdlr = None
        #if type == True:
        #  hdlr = SysLogHandler(address='/dev/log',
        #  facility=SysLogHandler.LOG_DAEMON)
        #else:
        hdlr = StreamHandler(sys.stdout)
        hdlr.setLevel(logging.INFO)
        hdlr.setFormatter(formatter)
        ROOTLOGGER.addHandler(hdlr)
        ROOTLOGGER.setLevel(logging.INFO)

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
        TideGate.init_logs()
        args = TideGate.parse_arguments(raw_args)
        client = Client(base_url=args.docker)
        start_time = datetime.utcnow()
        storage = core.ServerStorage()
        loader = ServerLoader(client,storage)
        loader.load_all()
        
        generator = NginxReverseProxyConfigGenerator("/tmp/test.config")
        generator.generate(storage) 
        processor = DockerEventProcessor(client, storage, loader)
        thread = DockerEventMonitorThread(processor,client,start_time)
        thread.start()
        thread.join()
        
class ServerLoader:
  def __init__(self,client,storage):
    self.client = client
    self.storage = storage
    
  def load_all(self):
    for c in self.client.containers(filters={"status":"running"}):
      server = self.load_server(c["Id"])
      if server != None:
        self.storage.add(server)
    
  def load_server(self,container_id):
    res = None
    try:
      infos = self.client.inspect_container(container = container_id)
      config  = infos["Config"] if ("Config" in infos and isinstance(infos["Config"],dict)) else None 
      network_settings  = infos["NetworkSettings"] if ("NetworkSettings" in infos and isinstance(infos["NetworkSettings"],dict)) else None
      if config != None  and network_settings != None:
        labels =  config["Labels"] if ("Labels" in config and isinstance(config["Labels"],dict)) else None
        if labels != None:
          ports = network_settings["Ports"] if  ("Ports" in network_settings and isinstance(network_settings["Ports"],dict)) else None
          tide_desc =  labels["tidegate_descriptor"] if  ("tidegate_descriptor" in labels and isinstance(labels["tidegate_descriptor"],str)) else None 
          if tide_desc != None and ports != None:
            descriptor = json.loads(tide_desc)
            if descriptor["domain"] :
              domain = descriptor["domain"] if "domain" in descriptor else None
              ssl_enabled = ("ssl" in descriptor) 
              for k,i in ports.items():
                container_port,container_port_type = k.split("/")
                host_port = i[0]["HostPort"] if (len(i) == 1 and "HostPort" in i[0]) else None
                host_ip = i[0]["HostIp"] if (len(i) == 1 and "HostIp" in i[0]) else None
                
                #if not self.storage.contains(domain, container_port) :
                res = core.Server(domain,container_port,ssl_enabled)
                #  self.storage.add(server)
                #else:
                #server = self.storage.get(domain, container_port)
                res.endpoints.append(core.Endpoint(host_ip,host_port))
                ROOTLOGGER.info(str(res))
    except Exception as e :
      print(e)
    return res


if __name__ == "__main__":
    sys.exit(TideGate.main(sys.argv[1:]))

