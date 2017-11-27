# SanFran: Serverless Functions for Kubernetes

Run your Javascript functions on Kubernetes in a high performance serverless engine.  Functions will only use resources when handling requests. Designed to minimize cold-start latency and scale horizontally for high traffic functions.

- Runs any JavaScript function (with npm modules)
- Very fast function spin up under 30ms
- Minimize cold-start latency with pooling + recycling
- Per function horizontal scaling
- SanFran itself can horizontally scale for larger deployments
- Create, update, delete functions instantly
- Easy to deploy on Kubernetes

## The SanFran Technology Stack

- SanFran :heart: JavaScript: Designed ground up to be the best and fastest serverless engine to run your Javascript functions on. Uses NodeJS and supports NPM module dependencies.

- SanFran :heart: Kubernetes: SanFran is Kubernetes-native. It's built entirely on Kubernetes with a custom controller to manage function spin-up and shutdown.

## Quickest Start

Provided you have a Kubernetes cluster and Helm installed on it.

```console
$ helm install https://raw.githubusercontent.com/dosco/sanfran/master/helm-chart/sanfran-0.1.0.tgz
```

The SanFran API can then be accessed as follows

```console
open http://your_ingress_ip/api/
```

## Quickstart

The easiest way to get started with SanFran is to install it using [Helm](https://github.com/kubernetes/helm) on [Minikube](https://github.com/kubernetes/minikube). Helm is a tool to help install apps on Kubernetes and Minikube is a small development version of Kubernetes.

### 1. Install and Setup Minikube

[Minikube & Helm Quick Setup](#minikube-setup-guide)

### 2. Download, compile and deploy SanFran

Install Go Dependency Manager Glide

```console
$ go get github.com/Masterminds/glide
```

Then compile and install SanFran

```console
$ go get https://github.com/dosco/sanfran.git
$ cd $GOPATH/src/github.com/dosco/sanfran
$ brew install jq
$ glide install
$ make docker
$ helm install ./helm-chart/sanfran/
```

## Fun With Functions

To add your JS function, use the `node cli/index.js` command. As an example run these commands to add a function that just returns http request headers.

```console
$ node cli/index.js
  ____                    _____
 / ___|    __ _   _ __   |  ___|  _ __    __ _   _ __
 \___ \   / _` | | '_ \  | |_    | '__|  / _` | | '_ \
  ___) | | (_| | | | | | |  _|   | |    | (_| | | | | |
 |____/   \__,_| |_| |_| |_|     |_|     \__,_| |_| |_|

? Sanfran API server: (Use arrow keys)
❯ Minikube IP
  Other
```

Or try adding the `demo/headers.js` or `demo/hello.js` sample functions

```console
? Sanfran API server: Minikube IP
? Pick an action you want to take: Create
? Enter function name: hello
? Enter filename of the function code: demo/hello.js
> https://192.168.64.4/fn/hello
```

Then go try the function see it running with your web browser

```console
$ open http://$(minikube ip)/fn/hello?name=Vik
```

## SanFran JS Functions

The Javascript functions that work with SanFran are just standard NodeJS HTTP functions. If you've worked in nodejs with packages like express you're very familiar with these types of functions. Here's the `hello.js` function

```javascript
module.exports = function (req, res) {
  res.send(`Hello ${req.query.name || 'World'}`);
};
```

While SanFran seems pretty simple, underneath it is designed to be a scalable and high performance engine to run your functions on.

## Performance Benchmarks

I use [Vegeta](https://github.com/tsenart/vegeta) a HTTP load testing tool and library for benchmarking cold-start performance on a warmed up Minikube. Initial basic testing has shown that our design provides the high performance we aim for with this project.

```console
$ echo "GET http://10.0.0.170/fn/headers?a=hello&b=world" | vegeta attack -duration=15s | vegeta report -reporter='hist[6ms,8ms,10ms,15ms,20ms]'
Bucket         #   %       Histogram
[6ms,   8ms]   81  32.40%  ########################
[8ms,   10ms]  67  26.80%  ####################
[10ms,  15ms]  53  21.20%  ###############
[15ms,  20ms]  13  5.20%   ###
[20ms,  +Inf]  36  14.40%  ##########
```

## Architecture Design in Brief

- Uses micro services: Router, Controller, API and Function Sidecar
- GRPC for efficient communication between services
- Client side GRPC load balancing between internal micro-services
- BoltDB for internal datastore
- Autoscaler for maintaining a pool of ready pods
- Pods activated with functions are downgraded and recycled when not in use
- All micro-services can be scaled horizontally for large deployments
- Keep the design simple and focus on performance, efficiency and extensibility

### Minikube Setup Guide

Minikube is a tool that makes it easy to run Kubernetes locally. Minikube runs a single-node Kubernetes cluster inside a VM on your laptop for users looking to try out Kubernetes or develop with it day-to-day.

#### Install Minikube (On Mac)

```console
$ brew cask install minikube
$ brew install docker-machine-driver-xhyve
$ sudo chown root:wheel $(brew --prefix)/opt/docker-machine-driver-xhyve/bin/docker-machine-driver-xhyve
$ sudo chmod u+s $(brew --prefix)/opt/docker-machine-driver-xhyve/bin/docker-machine-driver-xhyve
```

#### Start Minikube

```console
$ minikube start --vm-driver=xhyve
$ minikube addons enable ingress
$ eval $(minikube docker-env)
```

### Install Helm (Kubernetes Package Manager)

```console
$ brew install kubernetes-helm
$ helm init
```

### Minikube Development Tips

This part is not required but helps with debugging if you do development work
using Minikube. To make the IP's inside the Minikube cluster / vm accessible for your development host (Mac Laptop). The below commands will setup MacOS routing to allow for this.

```console
$ sudo route -n add 10.0.0.0/24 $(minikube ip)
$ sudo route -n add 172.17.0.0/16 $(minikube ip)
$ sudo ifconfig bridge100 -hostfilter $(ifconfig 'bridge100' | grep member | awk '{print $2}')
```

And then add the MiniKube DNS to your host.

```console
$ sudo mkdir /etc/resolver/
$ sudo dd of=/etc/resolver/svc.cluster.local <<EOF
nameserver 10.0.0.10
domain svc.cluster.local
search svc.cluster.local default.svc.cluster.local
options ndots:5
EOF
$ sudo defaults write /Library/Preferences/com.apple.mDNSResponder.plist AlwaysAppendSearchDomains -bool YES
```

Finally there are a couple ways to check if its all setup ok.

```
$ scutil --dns | grep "10.0.0.10" |  if [ $? -eq 0 ]; then echo "dns setup ok"; else echo "dns setup failed"; fi
```

And if you already have SanFran deployed on Minikube then use ping to see if its reachable.

```
$ ping sanfran-router-service
```

### Building and Developing on SanFran

For local development I use Minikube so all the below steps will require it to be installed and running. [Minikube Github](https://github.com/kubernetes/minikube)

SanFran is written entire in GO and depends on Kubernetes among other things. All dependencies are vendored in using the `glide` tool. Just use the command `make` to build all the services and cli tool natively on your dev box. To build and deploy the SanFran containers to your local Minikube instance use the below commands.

```console
$ eval $(minikube docker-env)
$ make docker
```

You can also build and run specific services with the below command.

```console
$ cd router
$ make run
```

## SanFran :heart: Developers

We are working on a developers guide. So until then just get started and if you have questions feel free to reach out on Twitter [twitter.com/dosco](https://twitter.com/dosco)

SanFran is well-tested on Minikube
