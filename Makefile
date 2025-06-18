# Variables
CLI_BINARY_NAME=tunman
DAEMON_BINARY_NAME=$(CLI_BINARY_NAME)d
BUILD_DIR=bin
EXT=$(if $(filter windows,$(GOOS)),.exe,)

# Targets
.PHONY: all build/* test release install clean lint

all: build/$(CLI_BINARY_NAME) build/$(DAEMON_BINARY_NAME)

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
