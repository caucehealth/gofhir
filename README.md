# gofhir

A comprehensive, zero-dependency Go library for the HL7 FHIR R4 specification. 145 resource types generated from the official schema, with a fluent API, validation engine, FHIRPath evaluator, REST client, SMART on FHIR auth, bulk data support, and more.

874/874 HL7 official examples pass JSON and XML round-trip. 2,400+ tests.

## Installation

```
go get github.com/caucehealth/gofhir
```

## Quick Start

```go
import (
    "encoding/json"
    "github.com/caucehealth/gofhir/r4/resources"
)

// Build a Patient
p, _ := resources.NewPatient().
    WithName("John", "Doe").
    WithBirthDate("1980-03-15").
    WithGender(resources.AdministrativeGenderMale).
    Build()

// Serialize to JSON
data, _ := json.Marshal(p)

// Parse from JSON
var patient resources.Patient
json.Unmarshal(data, &patient)
```

## Packages

| Package | Description |
|---------|-------------|
| [`r4/resources`](#resources) | 145 generated resource types with builders and interfaces |
| [`r4/datatypes`](#datatypes) | FHIR primitive and complex types with precision preservation |
| [`r4/parser`](#parsing) | JSON/XML parsing with 9 configurable options |
| [`r4/bundle`](#bundles) | Bundle builder and streaming iterator |
| [`r4/client`](#rest-client) | FHIR REST client with search, paging, and middleware |
| [`r4/validate`](#validation) | Schema, profile, slicing, and FHIRPath invariant validation |
| [`r4/fhirpath`](#fhirpath) | FHIRPath expression engine with 55+ functions |
| [`r4/terminology`](#terminology) | Code system validation (in-memory, remote, chain) |
| [`r4/smart`](#smart-on-fhir) | SMART on FHIR OAuth2 (standalone, EHR, backend services) |
| [`r4/bulk`](#bulk-data) | Bulk Data $export with NDJSON streaming |
| [`r4/patch`](#patch) | JSON Patch (RFC 6902) and FHIR Patch builders |
| [`r4/diff`](#diff) | Resource comparison and change detection |
| [`r4/synthetic`](#synthetic-data) | Random resource generation for testing |
| [`cmd/fhirlint`](#linter-cli) | CLI for validating FHIR resources |

---

## Resources

Every FHIR R4 resource is a native Go struct with JSON/XML round-trip support. All 145 resources implement the `Resource` interface for type-safe polymorphism.

```go
// Type-safe interface — no type assertions to any
type Resource interface {
    GetResourceType() string
    GetId() dt.ID
    GetMeta() dt.Meta
}

// Fluent builders for all resources
obs, _ := resources.NewObservation().
    WithStatus(resources.ObservationStatusFinal).
    WithCode("http://loinc.org", "8867-4", "Heart rate").
    WithSubject("Patient/123").
    Build()

// Polymorphic fields as tagged unions (not getter/setter explosion)
obs.Value = &resources.ObservationValue{
    Quantity: &dt.Quantity{Value: &decimal, Unit: &unit},
}
```

## Datatypes

Precision-preserving types for FHIR primitives:

```go
// Decimal preserves precision — no float64 rounding
d := dt.NewDecimal(3.14)  // marshals as 3.14, not 3.140000000000001

// Temporal types track precision
date := dt.Date("2024-03")       // year-month precision
date.Precision()                  // DatePrecisionMonth

// Extension registry with typed extraction
val, ok := dt.GetExtensionValue[string](patient.Extension, "http://example.org/ext")
```

## Parsing

JSON and XML parsing with configurable options. 874/874 HL7 official examples pass round-trip.

```go
import "github.com/caucehealth/gofhir/r4/parser"

// JSON with options
p := parser.New(
    parser.WithPrettyPrint(),
    parser.WithSuppressNarrative(),
    parser.WithSummaryMode(parser.SummaryTrue),
    parser.WithStrictMode(),
)
data, _ := p.Marshal(patient)

// XML round-trip
xmlData, _ := parser.MarshalXML(patient)
resource, _ := parser.UnmarshalXML(xmlData)
```

## Bundles

Builder pattern with streaming support for large bundles:

```go
import "github.com/caucehealth/gofhir/r4/bundle"

// Build a transaction bundle
b := bundle.New(bundle.TypeTransaction).
    WithTransactionEntry("POST", "Patient", patient).
    WithTransactionEntry("POST", "Observation", obs).
    Build()

// Stream large bundles with constant memory
iter := bundle.NewEntryIterator(reader)
for iter.Next() {
    entry := iter.Entry()
    // process entry.Resource
}
```

## REST Client

Full CRUD, search, history, paging, and operations with composable middleware:

```go
import "github.com/caucehealth/gofhir/r4/client"

c := client.New("https://hapi.fhir.org/baseR4")

// Type-safe reads with generics
p, _ := client.ReadAs[resources.Patient](ctx, c, "Patient", "123")

// Fluent search
results, _ := c.Search(ctx, "Patient").
    Where("family", "Smith").
    Where("birthdate", "gt2000-01-01").
    Count(10).
    Sort("birthdate").
    Execute()

// Version history
history, _ := client.History(ctx, c, "Patient", "123")
version, _ := client.VRead(ctx, c, "Patient", "123", "2")

// Patch
patchBody := patch.NewJSONPatch().Replace("/gender", "female").MustMarshal()
client.Patch(ctx, c, "Patient", "123", patchBody, patch.ContentTypeJSONPatch)

// Binary
data, contentType, _ := client.ReadBinary(ctx, c, "doc-1")

// Transactions
resp, _ := client.Transaction(ctx, c, txBundle)

// Middleware: auth, retry, logging, caching
c.Wrap(
    client.BearerToken("my-token"),
    client.Retry(3, time.Second),
    client.Logging(func(method, url string, status int, dur time.Duration) {
        log.Printf("%s %s → %d (%s)", method, url, status, dur)
    }),
    client.ETagCache(1000),
)
```

## Validation

Composable validation with schema checks, profile constraints, slicing, and FHIRPath invariants:

```go
import "github.com/caucehealth/gofhir/r4/validate"

// Built-in validation: required fields, enums, cardinality, primitive formats
v := validate.New()
result := v.Validate(observation)
for _, issue := range result.Errors() {
    fmt.Printf("%s: %s\n", issue.Path, issue.Message)
}

// Convert to OperationOutcome
oo := result.ToOperationOutcome()

// Profile validation with slicing support
registry := validate.NewProfileRegistry()
registry.Load(structureDefinitionJSON)
v = validate.New(validate.WithProfile(registry, "http://example.org/StructureDefinition/us-core-patient"))

// Custom rules
v = validate.New(validate.WithRules(validate.RuleFunc(func(r resources.Resource) []validate.Issue {
    // your custom validation logic
    return nil
})))

// FHIRPath invariants
v = validate.New(validate.WithInvariants(map[string]string{
    "obs-6": "value.exists() or dataAbsentReason.exists()",
}))

// Validate raw JSON
result, _ := validate.ValidateJSON(jsonBytes)
```

## FHIRPath

Full FHIRPath expression engine with 55+ built-in functions:

```go
import "github.com/caucehealth/gofhir/r4/fhirpath"

// Evaluate expressions
results, _ := fhirpath.Evaluate(patient, "name.where(use='official').family")
active, _ := fhirpath.EvaluateBool(patient, "active = true")

// Date arithmetic
fhirpath.Evaluate(patient, "birthDate + 18 'years'")

// Quantity comparison
fhirpath.EvaluateBool(obs, "value.where(value > 120 'mmHg').exists()")

// Register custom functions
fhirpath.RegisterFunction("myCheck", func(input []any, args []any) ([]any, error) {
    // custom logic
    return result, nil
})
```

## Terminology

Code system validation with pluggable backends:

```go
import "github.com/caucehealth/gofhir/r4/terminology"

// In-memory: 19 pre-loaded code systems
mem := terminology.NewInMemory()
valid, _ := mem.ValidateCode(ctx, "http://hl7.org/fhir/administrative-gender", "male")

// Remote: delegates to a FHIR terminology server
remote := terminology.NewRemote(fhirClient, "https://tx.fhir.org/r4")

// Chain: cascades through multiple services
chain := terminology.NewChain(mem, remote)
```

## SMART on FHIR

Complete SMART on FHIR authorization: standalone launch, EHR launch, backend services, PKCE, token refresh.

```go
import "github.com/caucehealth/gofhir/r4/smart"

// Discover endpoints
cfg, _ := smart.Discover(ctx, "https://fhir.example.org")

// Standalone launch
launcher := smart.NewStandaloneLauncher(cfg, smart.ClientConfig{
    ClientID:    "my-app",
    RedirectURI: "http://localhost:8080/callback",
    Scopes:      []string{"patient/Patient.read", "launch/patient"},
})
authURL := launcher.AuthURL("state-123")
// ... redirect user, handle callback ...
token, _ := launcher.Exchange(ctx, code)

// Backend services (client credentials with JWT)
token, _ := smart.BackendAuth(ctx, cfg, smart.BackendConfig{
    ClientID:   "service-app",
    PrivateKey: pemBytes,
    Scopes:     []string{"system/*.read"},
})

// Auto-refreshing token source
src := smart.NewRefreshingTokenSource(cfg, "my-app", secret, initialToken)
tok, _ := src.Token(ctx) // refreshes automatically when expired
```

## Bulk Data

FHIR Bulk Data Access ($export) with NDJSON streaming:

```go
import "github.com/caucehealth/gofhir/r4/bulk"

exporter := bulk.NewExporter("https://fhir.example.org")

// Kick off system-level export
job, _ := exporter.SystemExport(ctx, &bulk.ExportParams{
    Types: []string{"Patient", "Observation"},
})

// Wait for completion
status, _ := job.WaitForComplete(ctx)

// Stream NDJSON results
for _, output := range status.Output {
    reader, _ := job.Download(ctx, output)
    defer reader.Close()
    for reader.Next() {
        resource, _ := reader.Resource()
        fmt.Println(resource.GetResourceType(), resource.GetId())
    }
}

// Write NDJSON
writer := bulk.NewNDJSONWriter(file)
writer.Write(patient)
```

## Patch

JSON Patch (RFC 6902) and FHIR Patch builders:

```go
import "github.com/caucehealth/gofhir/r4/patch"

// JSON Patch
p := patch.NewJSONPatch().
    Test("/gender", "male").
    Replace("/gender", "female").
    Add("/telecom/-", map[string]string{"system": "phone", "value": "555-1234"}).
    Remove("/address/0")
data, _ := p.Marshal()

// FHIR Patch (Parameters resource)
fp := patch.NewFHIRPatch().
    Replace("Patient.birthDate", "1990-01-01").
    Add("Patient", "active", true).
    Delete("Patient.telecom")
data, _ := fp.Marshal()
```

## Diff

Compare two resource versions:

```go
import "github.com/caucehealth/gofhir/r4/diff"

result, _ := diff.Compare(oldPatient, newPatient)
for _, change := range result.Changes {
    fmt.Printf("%s %s: %v -> %v\n", change.Type, change.Path, change.OldValue, change.NewValue)
}
// Also works with raw JSON
result, _ = diff.CompareJSON(oldJSON, newJSON)
```

## Synthetic Data

Generate random FHIR resources for testing:

```go
import "github.com/caucehealth/gofhir/r4/synthetic"

gen := synthetic.New()                    // random seed
gen := synthetic.NewWithSeed(42)          // reproducible

patient := gen.Patient()                  // random patient with name, address, identifiers
obs := gen.Observation("patient-1")       // random vital sign
cond := gen.Condition("patient-1")        // random condition

// Generate a populated dataset
patients, observations, conditions, encounters := gen.PopulatedBundle(100)
```

## Linter CLI

Validate FHIR resources from the command line:

```bash
# Install
go install github.com/caucehealth/gofhir/cmd/fhirlint@latest

# Validate files
fhirlint patient.json observation.json

# Validate from stdin
cat bundle.json | fhirlint -

# JSON output for CI
fhirlint -format json *.json

# Quiet mode (errors only)
fhirlint -q patient.json
```

Exit codes: `0` = valid, `1` = validation errors, `2` = usage/IO error.

## Design

- **Zero dependencies** — only the Go standard library
- **Code generated** — all 145 resources generated from the official FHIR R4 JSON schema, reproducible with `go generate`
- **Multi-version ready** — all R4 types live under `r4/`; R5 will live under `r5/` with zero breaking changes
- **Type-safe** — generics, interfaces, and tagged unions instead of reflection and `any`
- **Composable** — functional options, middleware chains, rule interfaces

### Type Mappings

| FHIR Type | Go Type |
|---|---|
| string | `string` |
| boolean | `bool` |
| integer | `int32` |
| decimal | `datatypes.Decimal` (precision-preserving) |
| uri | `datatypes.URI` |
| code | `datatypes.Code` |
| date | `datatypes.Date` |
| dateTime | `datatypes.DateTime` |
| instant | `datatypes.Instant` |
| id | `datatypes.ID` |
| positiveInt | `uint32` |
| unsignedInt | `uint32` |
| base64Binary | `datatypes.Base64Binary` |

## Code Generation

```bash
make          # download schema + generate + build + test + vet
make schema   # download fhir.schema.json
make generate # run code generator
make test     # run tests
make examples # download HL7 official examples
```

## Trademark Notice

HL7 and FHIR are registered trademarks of Health Level Seven International.

## License

Apache 2.0 — see [LICENSE](LICENSE).
