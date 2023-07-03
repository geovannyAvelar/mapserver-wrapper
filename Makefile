BINARY_FILE_NAME=mapserver-wrapper
MAIN_FILE_PATH=main.go

build:
	go build -o ${BINARY_FILE_NAME} ${MAIN_FILE_PATH}

bootstrap:
	cp .env.example .env
	make load-env

load-env:
	source .env.example

run:
	go run ${MAIN_FILE_PATH}

clean:
	go clean
	rm -f ${BINARY_FILE_NAME} *.out