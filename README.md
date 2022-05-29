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

- [ ] Common
    - [ ] [Complex resolving (`$id`, `$anchor`, etc)](https://json-schema.org/understanding-json-schema/structuring.html)
    - [ ] [String formats](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00#section-7)
    - [ ] [ECMA 262 Regex](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00#section-3.3)
- [ ] [Draft 4](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00)
    - [x] [Validation keywords for numeric instances (number and integer)](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00#section-5.1)
        - [x] [multipleOf](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00#section-5.1.1)
        - [x] [maximum and exclusiveMaximum](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00#section-5.1.2)
        - [x] [minimum and exclusiveMinimum](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00#section-5.1.3)
    - [x] [Validation keywords for strings](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00#section-5.2)
        - [x] [maxLength](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00#section-5.2.1)
        - [x] [minLength](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00#section-5.2.2)
        - [x] [pattern](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00#section-5.2.3)
    - [x] [Validation keywords for arrays](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00#section-5.3)
        - [x] [additionalItems and items](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00#section-5.3.1)
          - [ ] `items` as array
        - [x] [maxItems](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00#section-5.3.2)
        - [x] [minItems](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00#section-5.3.3)
        - [x] [uniqueItems](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00#section-5.3.4)
    - [x] [Validation keywords for objects](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00#section-5.4)
        - [x] [maxProperties](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00#section-5.4.1)
        - [x] [minProperties](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00#section-5.4.2)
        - [x] [required](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00#section-5.4.3)
        - [x] [additionalProperties, properties and patternProperties](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00#section-5.4.4)
        - [ ] [dependencies](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00#section-5.4.5)
    - [x] [Validation keywords for any instance type](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00#section-5.5)
        - [x] [enum](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00#section-5.5.1)
        - [x] [type](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00#section-5.5.2)
        - [x] [allOf](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00#section-5.5.3)
        - [x] [anyOf](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00#section-5.5.4)
        - [x] [oneOf](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00#section-5.5.5)
        - [x] [not](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00#section-5.5.6)
        - [ ] [definitions](https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00#section-5.5.7)
- [ ] Draft 6
- [ ] Draft 7
- [ ] Draft 2019-09 / Draft 8
- [ ] Draft 2020-12
- [ ] Old drafts
    - [ ] Draft 1
    - [ ] Draft 2
    - [ ] Draft 3
