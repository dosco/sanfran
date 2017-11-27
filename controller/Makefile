image = dosco/sanfran-controller:0.2

default: rpc.pb.go
	go build -o build/sanfran-controller .

default.linux: rpc.pb.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/sanfran-controller .

run: rpc.pb.go
	go run *.go  -logtostderr -kubeconfig=$(HOME)/.kube/config

rpc.pb.go:
	protoc rpc/rpc.proto --gofast_out=plugins=grpc:.

docker: default.linux
	docker build -t $(image) .

docker-push:
	docker push $(image)