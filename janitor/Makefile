image = dosco/sanfran-janitor:0.4

default:
	go build -o build/sanfran-janitor .

default.linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/sanfran-janitor .

run:
	go run *.go  -logtostderr -kubeconfig=$(HOME)/.kube/config

docker: default.linux
	docker build -t $(image) .

docker-push:
	docker push $(image)