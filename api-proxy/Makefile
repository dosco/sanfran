image = dosco/sanfran-api-proxy:0.1

default:
	cp ../fnapi/rpc/rpc.swagger.json .
	go build -o build/sanfran-api-proxy .

default.linux:
	cp ../fnapi/rpc/rpc.swagger.json .
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/sanfran-api-proxy .

run:
	go run *.go -logtostderr

docker: default.linux
	docker build -t $(image) .

docker-push:
	docker push $(image)

bindata:
	cd swagger-ui && go-bindata-assetfs -nometadata -pkg swaggerUI dist/...
