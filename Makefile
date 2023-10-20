ROOT=$(shell pwd)
BUILD=$(ROOT)/build
BIN_NAME=langhelper
ifndef DOCKER_USER
	DOCKER_USER=sinashk
endif

build: clean
	mkdir $(BUILD)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o $(BUILD)/$(BIN_NAME) -ldflags="-extldflags=-static" -tags sqlite_omit_load_extension

run: build
	$(ROOT)/build/$(BIN_NAME)

docker: build
	docker build -t $(DOCKER_USER)/$(BIN_NAME):latest .

clean:
	rm -rf $(BUILD)
