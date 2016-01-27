#!/usr/bin/env python3
from abc import abstractmethod
from quik import Template
import json
import airspeed

class ReverseProxyConfigGenerator:
  
  def __init__(self):
    pass
  
  @abstractmethod
  def generate_configuration(self,services):
    pass


class NginxReverseProxyConfigGenerator(ReverseProxyConfigGenerator):
  def __init__(self, output_file_path):
    self.output_file_path = output_file_path
    
  def generate(self, servers):
    template = airspeed.Template("""
#foreach ($server in $servers)
upstream $server.domain.replace(".","_") {
  least_conn;
#foreach($endpoint in $server.endpoints)  server $endpoint.ip:$endpoint.port max_fails=3 fail_timeout=60 weight=1;
#end
}

server {
   listen $server.external_port;
   server_name $server.domain;
   charset utf-8;
 
   location / {
     proxy_pass http://$server.domain.replace(".","_")/;
     proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
     proxy_set_header Host $host;
     proxy_set_header X-Real-IP $remote_addr;
     proxy_set_header X-Forwarded-Proto $scheme;
   }


  #if($server.ssl)
    ssl on;
    ssl_certificate      /etc/letsencrypt/live/$server.domain/fullchain.pem;
    ssl_certificate_key  /etc/letsencrypt/live/$server.domain/privkey.pem;
    ssl_session_cache  builtin:1000  shared:SSL:10m;
    ssl_protocols  TLSv1 TLSv1.1 TLSv1.2;
    ssl_ciphers HIGH:!aNULL:!eNULL:!EXPORT:!CAMELLIA:!DES:!MD5:!PSK:!RC4;
    ssl_prefer_server_ciphers on;
  #end
 }
#end
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
""")
    print(json.dumps(servers))
    print(template.merge({"servers":json.loads(json.dumps(servers))}))
    #output_file = open(self.output_file_path,'w')
    #output_file.write(template.render(json.dumps(services)))
    #output_file.close()
    
  