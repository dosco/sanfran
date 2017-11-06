default: rpc.pb.go
	go build -o build/sanfran-sidecar .

default.linux: rpc.pb.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/sanfran-sidecar .

run: rpc.pb.go
	go run *.go -logtostderr

rpc.pb.go:
	protoc rpc/rpc.proto 	--gofast_out=plugins=grpc:.

docker: default.linux
	docker build -t sanfran-sidecar:v1 .

