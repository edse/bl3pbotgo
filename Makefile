build-go:
	CGO_ENABLED=0 GOOS=linux go build .

build: build-go
	docker-compose build --force-rm

stop:
	docker-compose down

run: stop
	docker-compose up
