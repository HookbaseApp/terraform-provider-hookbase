default: build

build:
	go build -o terraform-provider-hookbase

install: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/hookbase/hookbase/0.1.0/linux_amd64
	cp terraform-provider-hookbase ~/.terraform.d/plugins/registry.terraform.io/hookbase/hookbase/0.1.0/linux_amd64/

test:
	go test ./... -short -v

testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

lint:
	golangci-lint run ./...

fmt:
	gofmt -s -w .

tidy:
	go mod tidy

clean:
	rm -f terraform-provider-hookbase

.PHONY: default build install test testacc lint fmt tidy clean
