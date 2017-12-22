image = dosco/sanfran-fnapi:0.4

default: rpc.pb.go fnapi.swagger.json data.pb.go
	go build -o build/sanfran-fnapi .

default.linux: rpc.pb.go fnapi.swagger.json data.pb.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/sanfran-fnapi .

run: rpc.pb.go rpc.pb.gw.go data.pb.go
	go run *.go -logtostderr

rpc.pb.go:
	protoc -I/usr/local/include -I. \
	-I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
	--gofast_out=plugins=grpc:. \
	rpc/rpc.proto

rpc.pb.gw.go:
	protoc -I/usr/local/include -I. \
  -I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
  --grpc-gateway_out=logtostderr=true:. \
	--swagger_out=logtostderr=true:. \
  rpc/rpc.proto

fnapi.swagger.json: rpc.pb.gw.go
	jq -s '.[0] * .[1]'	\
	rpc/rpc.swagger.json rpc/rpc.swagger.patch.json \
	> rpc/fnapi.swagger.json

data.pb.go:
	protoc -I/usr/local/include -I. \
	--gofast_out=plugins=grpc:. \
	data/data.proto

docker: default.linux
	docker build -t $(image) .

docker-push:
	docker push $(image)