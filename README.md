goharproxy
=================
Alternative to [browsermob-proxy](https://github.com/lightbody/browsermob-proxy) written in go.

[![Build Status](https://travis-ci.org/Hellspam/go-selenium-proxy.svg?branch=master)](https://travis-ci.org/Hellspam/go-selenium-proxy)

Features
--------

Supports creating new proxies, serving HAR logs, and remapping hosts.

- Create proxy: POST /proxy
  - Returns : ```{ "port": [portNumber] }```

- Get HAR: PUT /proxy/[portNumber]/har
  - Returns HAR log in json, and clears previous entries
  
- Remapping hosts: POST /proxy/[portNumber]/hosts
  - Expects json containing array of : ```{ "Host" : [oldHost], "NewHost" : [newHost] }```
  - Supports IP / host name

- Delete Proxy: DELETE /proxy/[portNumber]

Currently does not fill whole HAR - timings contain only timing between request start and response end.
