default:
	cd fnapi && make
	cd sidecar && make
	cd controller && make
	cd router && make

docker:
	cd hello-nodejs && npm install
	cd base-nodejs && npm install && make docker
	cd fnapi && make docker
	cd sidecar && make docker
	cd controller && make docker
	cd router && make docker
