# jsonschema

[![Go Reference](https://pkg.go.dev/badge/github.com/tdakkota/jsonschema.svg)](https://pkg.go.dev/github.com/tdakkota/jsonschema)
[![codecov](https://codecov.io/gh/tdakkota/jsonschema/branch/master/graph/badge.svg?token=DVH08RoQyx)](https://codecov.io/gh/tdakkota/jsonschema)

jsonschema is a [JSON Schema Draft 4](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00)
validator implementation

## Usage

```go
package example

import (
	"fmt"

	"github.com/tdakkota/jsonschema"
)

func Example() {
	schema, err := jsonschema.Parse([]byte(`{
      "type": "object",
      "properties": {
        "number": { "type": "number" },
        "street_name": { "type": "string" },
        "street_type": { "enum": ["Street", "Avenue", "Boulevard"] }
      }
    }`))
	if err != nil {
		panic(err)
	}

	if err := schema.Validate(
		[]byte(`{ "number": 1600, "street_name": "Pennsylvania", "street_type": "Avenue" }`),
	); err != nil {
		panic(err)
	}

	fmt.Println(schema.Validate([]byte(`{"number": "1600"}`)))
	// Output:
	// object: "number": string: type is not allowed
}
```

## Install

```
go install -v github.com/tdakkota/jsonschema@latest
```

## Roadmap

See [this issue](https://github.com/tdakkota/jsonschema/issues/4).
