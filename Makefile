SCHEMA_URL := https://hl7.org/fhir/R4/fhir.schema.json.zip
SCHEMA_PATH := internal/spec/fhir.schema.json
SCHEMA_ZIP := /tmp/fhir.schema.json.zip
EXAMPLES_URL := https://www.hl7.org/fhir/R4/examples-json.zip
EXAMPLES_DIR := r4/resources/testdata/hl7-examples
EXAMPLES_ZIP := /tmp/fhir-examples-json.zip

.PHONY: all generate build test vet lint clean schema examples

all: schema generate build test vet lint

schema: $(SCHEMA_PATH)

$(SCHEMA_PATH):
	@echo "Downloading FHIR R4 schema..."
	@curl -sL -o $(SCHEMA_ZIP) $(SCHEMA_URL)
	@unzip -o $(SCHEMA_ZIP) -d internal/spec/
	@rm -f $(SCHEMA_ZIP)

examples: $(EXAMPLES_DIR)

$(EXAMPLES_DIR):
	@echo "Downloading FHIR R4 examples..."
	@mkdir -p $(EXAMPLES_DIR)
	@curl -sL -o $(EXAMPLES_ZIP) $(EXAMPLES_URL)
	@unzip -q -o $(EXAMPLES_ZIP) -d /tmp/fhir-examples-raw/
	@cp /tmp/fhir-examples-raw/*.json $(EXAMPLES_DIR)/
	@rm -rf /tmp/fhir-examples-raw $(EXAMPLES_ZIP)
	@echo "Downloaded $$(ls $(EXAMPLES_DIR)/*.json | wc -l) examples"

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
