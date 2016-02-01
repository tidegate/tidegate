#!/usr/bin/env python3

from enum import Enum
from json import JSONEncoder

def _default(self, obj):
    return getattr(obj.__class__, "__json__", _default.default)(obj)

_default.default = JSONEncoder().default  # Save unmodified default.
JSONEncoder.default = _default # replacement

class Endpoint:
  def __init__(self,ip,port):
    self.ip = ip
    self.port = port
  
  def __repr__(self):
    return str(self.__json__())
 
  def __json__(self):
    return {"ip":self.ip,"port":self.port}
  
class Server:
  def __init__(self,domain,port,ssl_enabled):
    self.domain = domain
    self.external_port = port
    self.endpoints = []
    self.ssl_enabled = ssl_enabled
    
  def __repr__(self):
    return str(self.__json__())
  
  def __json__(self):
    return {'domain':self.domain,"endpoints": self.endpoints,"external_port":self.external_port,"ssl_enabled":self.ssl_enabled}

class ServiceState(Enum):
  ACTIVE = 1
  INACTIVE = 2
   
class ServerStorage:
  
  def __init__(self):
    self.servers = {}
    
  def add(self,server):
    server_id = "{}:{}".format(server.domain,server.external_port)
    if not self.contains(server.domain, server.external_port):
      self.servers[server_id] = server
    else:
      self.servers[server_id].endpoints += server.endpoints
      
  
      
  def get(self,domain, port):
    res  = None
    server_id = "{}:{}".format(domain,port)
    if server_id in self.servers:
      res = self.servers[server_id]
    return res
  
  def contains(self,domain,port):
    server_id = "{}:{}".format(domain,port)
    return server_id in self.servers
  
  def __repr__(self):
    return str(self.__json__())
 
  def __json__(self):
    return list(self.servers.values())

    