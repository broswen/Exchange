.PHONY: build clean deploy gomodgen

build: gomodgen
	export GO111MODULE=on
	env CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bin/exchange exchange/main.go
	env CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bin/exchangeevent exchangeevent/main.go

clean:
	rm -rf ./bin ./vendor go.sum

deploy: clean build
	sls deploy --verbose

gomodgen:
	chmod u+x gomod.sh
	./gomod.sh
