"""Deterministic network edge sample: nginx -T, HAProxy, ingress, a DNS zone, certificates, partners, firewall rules, a CDN in the cloud inventory."""
import os
D = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'sample') + '/'
os.makedirs(D, exist_ok=True)

open(D + 'nginx-config.txt', 'w', newline='').write('''# configuration file /etc/nginx/nginx.conf:
http {
    ssl_protocols TLSv1 TLSv1.1 TLSv1.2;
    upstream portal_app {
        server 10.0.1.11:8080;
        server 10.0.1.12:8080;
    }
    upstream reports_app {
        server 10.0.1.21:8080;
    }
    server {
        listen 80;
        server_name portal.harvest.example;
        return 301 https://$host$request_uri;
    }
    server {
        listen 443 ssl;
        server_name portal.harvest.example;
        ssl_certificate /etc/nginx/certs/portal.crt;
        ssl_protocols TLSv1 TLSv1.1 TLSv1.2;
        location / { proxy_pass http://portal_app; }
    }
    server {
        listen 8081;
        server_name reports.harvest.local;
        location / { proxy_pass http://reports_app; }
    }
}
''')

open(D + 'haproxy.cfg', 'w', newline='').write('''global
    ssl-default-bind-options ssl-min-ver TLSv1.2
frontend api_front
    bind *:443 ssl crt /etc/haproxy/api.pem
    default_backend api_back
backend api_back
    server api1 10.0.1.31:9000 check
    server api2 10.0.1.32:9000 check
''')

open(D + 'ingress.json', 'w', newline='').write('''{
  "apiVersion": "v1",
  "kind": "List",
  "items": [
    {
      "apiVersion": "networking.k8s.io/v1",
      "kind": "Ingress",
      "metadata": { "name": "grower-portal", "namespace": "harvest" },
      "spec": {
        "tls": [ { "hosts": ["growers.harvest.example"] } ],
        "rules": [ { "host": "growers.harvest.example", "http": { "paths": [ { "path": "/", "backend": { "service": { "name": "grower-portal-svc" } } } ] } } ]
      }
    }
  ]
}
''')

open(D + 'harvest.example.zone', 'w', newline='').write('''$ORIGIN harvest.example.
$TTL 86400
@        IN SOA ns1.harvest.example. hostmaster.harvest.example. ( 2026111701 3600 600 604800 300 )
portal   IN A     203.0.113.20
growers  300 IN CNAME harvest-fd.azurefd.net.
static   IN CNAME d111111abcdef8.cloudfront.net.
api      IN A     203.0.113.21
''')

open(D + 'certs.txt', 'w', newline='').write('''subject=CN = portal.harvest.example
issuer=C = US, O = Example Trust, CN = R3
notAfter=Jan 10 12:00:00 2024 GMT
Public-Key: (2048 bit)
subject=CN = api.harvest.example
issuer=C = US, O = Example Trust, CN = R3
notAfter=Mar  1 00:00:00 2035 GMT
Public-Key: (4096 bit)
''')

open(D + 'partners.csv', 'w', newline='').write('''partner,host,port,protocol,direction,owner
Partner Bank,sftp.partnerbank.example,22,SFTP,out,Treasury (J. Ortiz)
Grain Co-op,edi.graincoop.example,4080,AS2,both,
Old Haulier,ftp.haulier.example,21,FTP,in,Logistics
Regional Office,vpn.region.example,500,VPN,both,IT network team
''')

open(D + 'firewall-rules.csv', 'w', newline='').write('''name,source,destination,port,protocol,action,description
web-in,any,203.0.113.20,443,tcp,allow,Portal HTTPS
rdp-admin,any,10.0.1.31,3389,tcp,allow,
legacy-all,10.0.9.0/24,any,any,any,allow,Old migration rule
bank-sftp-out,10.0.1.31,198.51.100.10,22,tcp,allow,Payments to Partner Bank
deny-rest,any,any,any,any,deny,Default deny
''')

open(D + 'azure_resources.csv', 'w', newline='').write('''name,type,location,resourceGroup
harvest-fd,microsoft.cdn/profiles,global,rg-harvest
harvest-vm,microsoft.compute/virtualmachines,westeurope,rg-harvest
''')

open(D + 'ipsec.conf', 'w', newline='').write('''conn regional-office
    left=203.0.113.1
    right=vpn.region.example
    authby=secret
    psk=RegionVpnKey2026!
''')
print('sample ok')
