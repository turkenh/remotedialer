Upbound Agent POC with Remotedialer
===================================

# Steps to follow

On main cluster
1. kubectl create ns tunnel-poc
2. cd charts/tunnel
3. helm upgrade --install tunnel-server . --set replicaCount=3

On crossplane cluster (tested with hosted crossplane instance whose ingress removed)
1. helm upgrade --install tunnel-client . --set client.id=53851874-df16-4a13-9119-f5c9657ebca2
2. create netpol to allow traffic from tunnel client to gateway & graphql