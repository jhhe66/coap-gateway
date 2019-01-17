[![Build Status](https://travis-ci.com/go-ocf/coap-gateway.svg?branch=master)](https://travis-ci.com/go-ocf/coap-gateway)
[![codecov](https://codecov.io/gh/go-ocf/coap-gateway/branch/master/graph/badge.svg)](https://codecov.io/gh/go-ocf/coap-gateway)
[![Go Report](https://goreportcard.com/badge/github.com/go-ocf/coap-gateway)](https://goreportcard.com/report/github.com/go-ocf/coap-gateway)

# coap-gateway

# Overview

OCF Servers / Clients communicate over TCP / UDP using the CoAP application protocol. Communication within the OCF Native Cloud shouldn't be restricted to the CoAP protocol, implementation should allow the use of whatever protocol might be introduced in the future. That's why the gateway is the access point for CoAP over TCP, and further communication is OCF Native Cloud specific.

TCP connection to the OCF Native Cloud is by its nature stateful. The OCF CoAP Gateway is therefore also stateful, keeping open connections to the OCF Servers / Clients.  The goal of the Gateway is to translate between the OCF Servers / Clients (CoAP) and the protocol of the OCF Native Cloud and communicate in an asynchronous way.

# Validation
- OCF CoAP Gateway can accept requests from the OCF Client / Server only after a successful sign-in 
- OCF CoAP Gateway can forward requests to the OCF Client / Server only after successful sign-in 
- If sign-in was not issued within the configured amount of time or sign-in request failed, OCF Native Cloud will forcibly close the TCP connection
- OCF CoAP Gateway sends command to update device core resource with its status.
  - Online when the device was successfully signed-in and communication lock released
  - Offline when the device was disconnected or signed-out
- Access Token from a successful sign-in must be locally persisted in the OCF CoAP Gateway and linked with an opened TCP channel
- Access Token linked with the opened TCP channel has to be included in each command issued to other OCF Native Cloud components
- OCF CoAP Gateway processes only those commands, which are designated for a device which the Gateway has an opened TCP channel to
- OCF CoAP Gateway is observing each resource published to the resource directory and publishes an event for every change
- OCF CoAP Gateway retrieves each published resource and updates Resources
- OCF CoAP Gateway has to expose the coap ping-pong + retry count configuration, which can be configured during the deployment
- OCF CoAP Gateway has to ping the device in the configured time, if pong is not received after the configured number of retries, then the connection with the device is closed and device is set as offline
- OCF CoAP Gateway processes events from Resources, by issuing a proper CoAP request to the device and raising an event with the response
- OCF CoAP Gateway has to process a waiting request within the configured time, or set the device as offline