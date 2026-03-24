# gofhir

A clean, fully typed Go library that implements the FHIR R4 specification. Every resource is represented as a native Go struct with JSON marshal/unmarshal support and a fluent builder API.

Generated from the official [HL7 FHIR R4 JSON schema](https://hl7.org/fhir/R4/).

## Installation

```
go get github.com/caucehealth/gofhir
```

## Usage

```go
import (
    "encoding/json"
    "fmt"

    "github.com/caucehealth/gofhir/r4/resources"
    "github.com/caucehealth/gofhir/r4/bundle"
)

// Parse a FHIR Patient from JSON
var patient resources.Patient
err := json.Unmarshal(data, &patient)

// Build a Patient using the fluent builder
p, err := resources.NewPatient().
    WithName("John", "Doe").
    WithBirthDate("1980-03-15").
    WithGender(resources.AdministrativeGenderMale).
    Build()

// Serialize to JSON
out, err := json.Marshal(p)

// Wrap in a Bundle
b := bundle.New(bundle.TypeSearchset).
    WithEntry(p).
    Build()
bundleJSON, err := json.Marshal(b)
```

## Features

- **Full FHIR R4 coverage** — 146 resources generated from the official schema
- **Strongly typed** — all fields, references, and code bindings are typed
- **JSON round-trip** — full marshal/unmarshal with correct `omitempty` handling
- **Fluent builders** — construct resources with a chainable API
- **Polymorphic fields** — value[x] fields handled as tagged union structs
- **Extensions** — nested extensions supported on all types
- **Enums** — required ValueSet bindings become Go enum types
- **GoDoc** — every exported symbol has documentation sourced from the FHIR specification
- **Zero dependencies** — only the Go standard library

## Package Structure

```
gofhir/
  cmd/gen/            # Code generator binary
  r4/
    datatypes/        # FHIR primitive and complex types
    resources/        # One file per resource (generated)
    bundle/           # Bundle handling
  internal/spec/      # Schema loader (generator only)
```

## Code Generation

The types are generated from the official FHIR R4 JSON schema. Use `make` to download the schema, generate code, build, test, and lint:

```bash
make          # download schema + generate + build + test + vet + lint
make schema   # download fhir.schema.json only
make generate # run code generator
make test     # run tests
```

Or run the generator directly (requires the schema to already be downloaded):

```bash
go generate ./...
```

## Design

This library is designed for future multi-version support. All R4 types live under `r4/` — when R5 support is added, it will live under `r5/` with zero breaking changes to R4 consumers.

### Type Mappings

| FHIR Type     | Go Type           |
|---------------|-------------------|
| string        | `string`          |
| boolean       | `bool`            |
| integer       | `int32`           |
| decimal       | `float64`         |
| uri           | `datatypes.URI`   |
| code          | `datatypes.Code`  |
| date          | `datatypes.Date`  |
| dateTime      | `datatypes.DateTime` |
| instant       | `datatypes.Instant` |
| id            | `datatypes.ID`    |
| positiveInt   | `uint32`          |
| unsignedInt   | `uint32`          |
| base64Binary  | `[]byte`          |

### Instant Handling

FHIR instants are RFC3339 timestamps with mandatory timezone offset. They are represented as `datatypes.Instant` (a `string` type alias) rather than `time.Time` to preserve round-trip fidelity. Use `datatypes.ParseInstant()` to convert to `time.Time` when needed.

## Trademark Notice

HL7® and FHIR® are registered trademarks of Health Level Seven International.

## License

Apache 2.0 — see [LICENSE](LICENSE).
