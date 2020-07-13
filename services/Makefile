.PHONY: build clean deploy

build:
#	dep ensure -v
	env GOOS=linux go build -ldflags="-s -w" -o bin/purl purl/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/epub2mobi epub2mobi/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/mailer mailer/main.go

clean:
	rm -rf ./bin ./vendor Gopkg.lock

deploy: clean build
	sls deploy --verbose
