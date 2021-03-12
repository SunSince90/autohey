# Update gomod
update-gomod:
	go mod download
	go mod tidy 
	go mod verify

# Build this
build:
	go build -a -o bin/autohey *.go

# Build the docker image
docker-build: test
	docker build . -t ${IMG}

# Push the docker image
docker-push:
	docker push ${IMG}