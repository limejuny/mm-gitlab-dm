BINARY_NAME=plugin

.PHONY: default
default: build

.PHONY: build
build:
	@echo "Building..."
	@cp const.go const.go.bak
	@cp const.go.cred const.go
	@go build -o bin/$(BINARY_NAME) -v
	@mv const.go.bak const.go
	@tar -czf bin/$(BINARY_NAME).tar.gz plugin.json -C bin/ $(BINARY_NAME)

# .PHONY: deploy
# deploy:
# 	@bin/mmctl plugin delete com.lguplus.nucube.nuap.mattermost-chatbot
# 	@bin/mmctl plugin add bin/$(BINARY_NAME).tar.gz
# 	@bin/mmctl plugin enable com.lguplus.nucube.nuap.mattermost-chatbot

.PHONY: clean
clean:
	@echo "Cleaning..."
	@rm -rf bin/${BINARY_NAME} bin/$(BINARY_NAME).tar.gz
