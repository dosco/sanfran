# SanFran: Serverless Functions for Kubernetes

Run your Javascript functions on Kubernetes in a high performance environment. The functions will only use resources when handling requests. Designed to minimize cold-start latency using instance pools and container recycling. Ability to scale horizontally for high QPS functions.

- JavaScript functions (with npm modules)
- Fast spin up (minimize cold-start with container pooling and recycling)
- Per function horizontal scaling
- Easy to deploy on Kubernetes

## The SanFran Technology Stack

- SanFran :heart: JavaScript: Designed ground up to be the best and fastest serverless engine to run your Javascript functions on. Uses NodeJS and supports NPM module dependencies.

- SanFran :heart: Kubernetes: SanFran is Kubernetes-native. It's built entirely on Kubernetes with a custom controller to manage function spin-up and shutdown.

## Quickstart

The easiest way to get started with SanFran is to install it using `kubectl` on Minikube. Minikube is the development version of Kubernetes and instructions of installing it are later in this document:

```console
$ git clone https://github.com/dosco/sanfran.git
$ cd sanfran
$ eval $(minikube docker-env)
$ make docker
$ kubectl apply -f sanfran.yaml
```

You will now have SanFran installed and deployed on Kubernetes

To add your JS function, use the `cli/build/sanfran-cli` command. As an example run these commands to add a function that just returns http request headers.

```console
$ cli/build/sanfran-cli create headers -file hello-nodejs/headers.js -host sanfran-fnapi-service
$ curl 'http://sanfran-routing-service/fn/headers?a=hello&b=world'
```

You can add as many functions as you like or use 'update', 'delete' or 'list' commands with
`cli/build/sanfran-cli` to manage existing functions. Here's another hello world example.

```console
$ cli/build/sanfran-cli create hello -file hello-nodejs/hello.js -host sanfran-fnapi-service
$ curl curl 'http://sanfran-routing-service/fn/hello?name=Vik'
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

I use [https://github.com/tsenart/vegeta](Vegeta) a HTTP load testing tool and library for benchmarking cold-start performance on a warmed up Minikube.

```console
$ echo "GET http://10.0.0.170/fn/headers?a=hello&b=world" | vegeta attack -duration=2s | tee results.bin | vegeta report -reporter='hist[6ms,20ms,30ms,40ms,50ms]'

[6ms,   20ms]  0   0.00%
[20ms,  30ms]  38  38.00%  ############################
[30ms,  40ms]  30  30.00%  ######################
[40ms,  50ms]  17  17.00%  ############
[50ms,  +Inf]  15  15.00%  ###########
```

## Architecture Design in Brief

- Uses micro services: Router, Controller, API and Function Sidecar
- GRPC for efficient communication between services
- BoltDB for internal datastore
- Autoscaler for maintaining a pool of ready pods
- Pods activated with functions are downgraded and recycled when not in use
- Keep the design simple and focus on performance, efficiency and extensibility

### Building and Developing on SanFran

For local development I use Minikube so all the below steps will require it to be installed and running. [https://github.com/kubernetes/minikube](Minikube Github)

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

### Minikube Development Tips

It helps to make the IP's inside the Minikube cluster / vm accessible for your development host (Mac Laptop). The below commands will setup MacOS routing to allow for this.

```console
$ sudo route -n add 10.0.0.0/24 $(minikube ip)
$ sudo route -n add 172.17.0.0/16 $(minikube ip)
$ sudo ifconfig bridge100 -hostfilter $(ifconfig 'bridge100' | grep member | awk '{print $2}’)
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
$ ping sanfran-routing-service
```

## SanFran :heart: Developers

We are working on a developers guide. So until then just get started and if you have questions feel free to reach out on Twitter [https://twitter.com/dosco](twitter.com/dosco)

SanFran is well-tested on Minikube
