default:
	cd cli && npm install
	cd fnapi && make
	cd builder && make
	cd sidecar && make
	cd controller && make
	cd router && make
	cd api-proxy && make
	cd janitor && make
	go build -o cli/build/sanfran-cli .

docker:
	cd cli && npm install
	cd demo && npm install
	cd base-nodejs && npm install && make docker
	cd fnapi && make docker
	cd builder && make docker
	cd sidecar && make docker
	cd controller && make docker
	cd router && make docker
	cd api-proxy && make docker
	cd janitor && make docker

docker-push:
	cd base-nodejs && npm install && make docker-push
	cd fnapi && make docker-push
	cd builder && make docker-push
	cd sidecar && make docker-push
	cd controller && make docker-push
	cd router && make docker-push
	cd api-proxy && make docker-push
	cd janitor && make docker-push

deploy:
	kubectl apply -f sanfran.yaml

undeploy:
	kubectl delete -f sanfran.yaml && kubectl delete pod -l 'app=sanfran-func'

