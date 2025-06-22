# Variables
CLI_BINARY_NAME=tunman
DAEMON_BINARY_NAME=$(CLI_BINARY_NAME)d
BUILD_DIR=bin
EXT=$(if $(filter windows,$(GOOS)),.exe,)
SERVICE_TEMPLATE_DIR=templates
# systemd is $(SERVICE_TEMPLATE_DIR)/systemd/tunmand.service

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
	@./scripts/escalate.sh cp ./$(BUILD_DIR)/$(CLI_BINARY_NAME)$(EXT) /usr/local/bin/$(CLI_BINARY_NAME)$(EXT)
	@./scripts/escalate.sh cp ./$(BUILD_DIR)/$(DAEMON_BINARY_NAME)$(EXT) /usr/local/bin/$(DAEMON_BINARY_NAME)$(EXT)

clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete."

lint:
	@./scripts/check-lint.sh

docs:
	@go run ./cmd/docs

install-user-service: install
	@echo "Installing user-level systemd service..."
	@mkdir -p ~/.config/systemd/user
	@mkdir -p ~/.config/tunman
	@cp $(SERVICE_TEMPLATE_DIR)/systemd/tunmand.service ~/.config/systemd/user/tunmand.service
	@systemctl --user daemon-reexec || true
	@systemctl --user daemon-reload
	@systemctl --user enable --now tunmand.service
	@echo "User-level systemd service installed and started."

uninstall-user-service:
	@echo "Uninstalling user-level systemd service..."
	@systemctl --user disable --now tunmand.service || true
	@rm -f ~/.config/systemd/user/tunmand.service
	@systemctl --user daemon-reload
	@echo "Removed user-level systemd service."

	@echo "Removing installed binaries..."
	@./scripts/escalate.sh rm -f /usr/local/bin/$(CLI_BINARY_NAME)$(EXT)
	@./scripts/escalate.sh rm -f /usr/local/bin/$(DAEMON_BINARY_NAME)$(EXT)
	@echo "Uninstallation complete."
