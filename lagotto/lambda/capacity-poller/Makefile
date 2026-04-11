.PHONY: build clean deploy

build:
	@echo "Building capacity-poller Lambda..."
	GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bootstrap main.go
	zip function.zip bootstrap
	@echo "Build complete: function.zip"

clean:
	rm -f bootstrap function.zip

deploy: build
	@echo "Deploying to Lambda..."
	aws lambda update-function-code \
		--function-name lagotto-capacity-poller \
		--zip-file fileb://function.zip
	@echo "Deploy complete"
