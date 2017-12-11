image = dosco/sanfran-api-proxy:0.3

fnapi.swagger.json:
	cp ../fnapi/rpc/fnapi.swagger.json .

default: fnapi.swagger.json
	go build -o build/sanfran-api-proxy .

default.linux: fnapi.swagger.json
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/sanfran-api-proxy .

run: fnapi.swagger.json
	go run *.go -logtostderr

docker: default.linux
	docker build -t $(image) .

docker-push:
	docker push $(image)

bindata:
	cd swagger-ui && go-bindata-assetfs -nometadata -pkg swaggerUI dist/...
