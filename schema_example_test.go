package jsonschema_test

import (
	"fmt"

	"github.com/tdakkota/jsonschema"
)

func ExampleParse() {
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
