# Variables
CLI_BINARY_NAME=tunman
DAEMON_BINARY_NAME=$(CLI_BINARY_NAME)d
BUILD_DIR=bin
EXT=$(if $(filter windows,$(GOOS)),.exe,)

# Targets
.PHONY: all proto build/* test release install clean lint docs

all: build/$(CLI_BINARY_NAME) build/$(DAEMON_BINARY_NAME)

proto:
	@protoc --proto_path=proto --go_out=proto --go-grpc_out=proto --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative ctrl.proto

build/%:
	@echo "Building $*..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$*$(EXT) ./cmd/$*
	@echo "Build complete: $(BUILD_DIR)/$*$(EXT)"

test:
	@go test ./...

release/%:
	@echo "Building the application..."
	@mkdir -p $(BUILD_DIR)
	@go build -mod=readonly -ldflags "-w -s" -o $(BUILD_DIR)/$*$(EXT) ./cmd/$*
	@echo "Build complete."

install: release
	@echo "installing"
	@./scripts/util/escalate.sh cp ./$(BUILD_DIR)/$(CLI_BINARY_NAME)$(EXT) /usr/local/bin/$(CLI_BINARY_NAME)$(EXT)
	@./scripts/util/escalate.sh cp ./$(BUILD_DIR)/$(DAEMON_BINARY_NAME)$(EXT) /usr/local/bin/$(DAEMON_BINARY_NAME)$(EXT)

clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete."

lint:
	@./scripts/check-lint.sh

docs:
	@go run ./cmd/docs
