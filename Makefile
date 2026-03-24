SCHEMA_URL := https://hl7.org/fhir/R4/fhir.schema.json.zip
SCHEMA_PATH := internal/spec/fhir.schema.json
SCHEMA_ZIP := /tmp/fhir.schema.json.zip

.PHONY: all generate build test vet lint clean schema

all: schema generate build test vet lint

schema: $(SCHEMA_PATH)

$(SCHEMA_PATH):
	@echo "Downloading FHIR R4 schema..."
	@curl -sL -o $(SCHEMA_ZIP) $(SCHEMA_URL)
	@unzip -o $(SCHEMA_ZIP) -d internal/spec/
	@rm -f $(SCHEMA_ZIP)

generate: schema
	go generate ./...

build: generate
	go build ./...

test: build
	go test ./...

vet: build
	go vet ./...

lint: build
	@which staticcheck > /dev/null 2>&1 || (echo "Installing staticcheck..." && go install honnef.co/go/tools/cmd/staticcheck@latest)
	staticcheck ./...

clean:
	rm -f $(SCHEMA_PATH)
	rm -f r4/resources/*_gen.go
	rm -f r4/datatypes/*_gen.go
