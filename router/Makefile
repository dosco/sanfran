image = dosco/sanfran-router:0.4

default:
	go build -o build/sanfran-router .

default.linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/sanfran-router .

run:
	go run *.go -kubeconfig=$(HOME)/.kube/config -logtostderr

docker: default.linux
	docker build -t $(image) .

docker-push:
	docker push $(image)