image = dosco/sanfran-nodejs:0.4

docker:
	npm install && npm cache clean --force
	docker build -t $(image) .

docker-push:
	docker push $(image)