gas-oracle:
	env GO111MODULE=on go build -v $(LDFLAGS) ./cmd/gas-oracle

clean:
	rm gas-oracle

test:
	go test -v ./...

lint:
	golangci-lint run ./...


.PHONY: \
	gas-oracle \
	clean \
	test \
	lint