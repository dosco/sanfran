image = dosco/sanfran-builder:0.3

default: rpc.pb.go
	go build -o build/sanfran-builder .

default.linux: rpc.pb.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/sanfran-builder .

run: rpc.pb.go
	SANFRAN_BUILDER_STORAGE_PATH=/tmp \
	SANFRAN_BUILDER_INPUT_PATH=/tmp \
	go run *.go -logtostderr

rpc.pb.go:
	protoc rpc/rpc.proto 	--gofast_out=plugins=grpc:.

docker: default.linux
	docker build -t $(image) .

docker-push:
	docker push $(image)

