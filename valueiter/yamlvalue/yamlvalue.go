package yamlvalue

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/go-faster/errors"
	"github.com/go-faster/jx"
	"github.com/go-faster/yaml"

	"github.com/tdakkota/jsonschema/valueiter"
)

var _ valueiter.Value[Value, string, string] = Value{}

// Value is valueiter.Value implementation for yaml.
type Value struct {
	Node *yaml.Node
}

func resolveNode(n *yaml.Node) (_ *yaml.Node, reason string) {
	if n == nil {
		return nil, "node is nil"
	}
	switch n.Kind {
	case yaml.DocumentNode:
		if len(n.Content) == 0 {
			return nil, "document node content is empty"
		}
		return resolveNode(n.Content[0])
	case yaml.AliasNode:
		return resolveNode(n.Alias)
	case yaml.MappingNode:
		if len(n.Content)%2 != 0 {
			return nil, "mapping node content length is not even"
		}
		fallthrough
	default:
		return n, ""
	}
}

func resolveNodeOr(n, fallback *yaml.Node) *yaml.Node {
	n, _ = resolveNode(n)
	if n == nil {
		return fallback
	}
	return n
}

func parseRat(value string) (*big.Rat, error) {
	rat, ok := new(big.Rat).SetString(value)
	if !ok {
		return nil, errors.Errorf("cannot unmarshal %q into *big.Rat", value)
	}
	return rat, nil
}

// Type implements valueiter.Value.
func (v Value) Type() jx.Type {
	n, _ := resolveNode(v.Node)
	if n == nil {
		return jx.Invalid
	}
	switch n.Kind {
	case yaml.MappingNode:
		return jx.Object
	case yaml.SequenceNode:
		return jx.Array
	case yaml.ScalarNode:
		switch n.Tag {
		case "!!null":
			return jx.Null
		case "!!bool":
			return jx.Bool
		case "!!int", "!!float":
			return jx.Number
		case "!!str":
			return jx.String
		default:
			// FIXME(tdakkota): is it correct?
			return jx.String
		}
	default:
		panic(fmt.Sprintf("unexpected node kind: %v", n.Kind))
	}
}

func decode[T any](v Value) (val T) {
	n := resolveNodeOr(v.Node, v.Node)
	return errors.Must(val, n.Decode(&val))
}

// Bool implements valueiter.Value.
func (v Value) Bool() bool {
	return decode[bool](v)
}

// Number implements valueiter.Value.
func (v Value) Number() valueiter.Number {
	n := resolveNodeOr(v.Node, v.Node)
	rat := errors.Must(parseRat(n.Value))
	return valueiter.Number{
		Rat: rat,
	}
}

// Str implements valueiter.Value.
func (v Value) Str() string {
	n := resolveNodeOr(v.Node, v.Node)
	return n.Value
}

// Array implements valueiter.Value.
func (v Value) Array(cb func(Value) error) error {
	n, reason := resolveNode(v.Node)
	if n == nil {
		return errors.Errorf("node is invalid: %s", reason)
	}
	for _, n := range n.Content {
		if err := cb(Value{Node: n}); err != nil {
			return err
		}
	}
	return nil
}

// Object implements valueiter.Value.
func (v Value) Object(cb func(key string, value Value) error) error {
	n, reason := resolveNode(v.Node)
	if n == nil {
		return errors.Errorf("node is invalid: %s", reason)
	}

	content := n.Content
	for i := 0; i < len(content); i += 2 {
		key := content[i]
		value := content[i+1]
		if err := cb(key.Value, Value{Node: value}); err != nil {
			return err
		}
	}
	return nil
}

var _ valueiter.ValueComparator[Value] = Comparator{}

// Comparator is Value comparator.
type Comparator struct{}

func (c Comparator) Contains(enum []json.RawMessage, val Value) (bool, error) {
	// FIXME(tdakkota): this is dramatically slow.
	for _, e := range enum {
		var n yaml.Node
		if err := yaml.Unmarshal(e, &n); err != nil {
			return false, err
		}

		ok, err := yamlEqual(val.Node, &n)
		if err != nil {
			return true, errors.Wrapf(err, "compare %q and %v", e, val)
		}

		if ok {
			return true, nil
		}
	}

	return false, nil
}

// Equal implements ValueComparator interface.
func (c Comparator) Equal(a, b Value) (bool, error) {
	return yamlEqual(a.Node, b.Node)
}

func yamlEqual(a, b *yaml.Node) (bool, error) {
	a, reason := resolveNode(a)
	if reason != "" {
		return false, errors.Errorf("left node is invalid: %s", reason)
	}
	b, reason = resolveNode(b)
	if reason != "" {
		return false, errors.Errorf("right node is invalid: %s", reason)
	}

	switch {
	case a == b:
		// Fast path check.
		return true, nil
	case a.Kind != b.Kind:
		// Ensure Kind is the same.
		return false, nil
	}

	switch a.Kind {
	case yaml.ScalarNode:
		if a.Value == b.Value && a.Tag == b.Tag {
			return true, nil
		}
		switch a.Tag {
		case "!!int", "!!float":
			switch b.Tag {
			case "!!int", "!!float":
			default:
				return false, nil
			}

			aRat, err := parseRat(a.Value)
			if err != nil {
				return false, errors.Wrap(err, "parse left number")
			}
			bRat, err := parseRat(b.Value)
			if err != nil {
				return false, errors.Wrap(err, "parse right number")
			}

			return aRat.Cmp(bRat) == 0, nil
		default:
			return false, nil
		}
	case yaml.SequenceNode:
		if len(a.Content) != len(b.Content) {
			return false, nil
		}
		for i := range a.Content {
			eq, err := yamlEqual(a.Content[i], b.Content[i])
			if err != nil {
				return false, errors.Wrapf(err, "compare [%d]", i)
			}
			if !eq {
				return false, nil
			}
		}
	case yaml.MappingNode:
		if len(a.Content) != len(b.Content) {
			return false, nil
		}
		validateMapping := func(n *yaml.Node) error {
			content := n.Content
			if l := len(content); l%2 != 0 {
				return errors.Errorf("mapping node should have even number of children, got %d", l)
			}
			for i := 0; i < len(content); i += 2 {
				if key := content[i]; key.Kind != yaml.ScalarNode {
					return errors.Errorf("key %q is not scalar", key.Value)
				}
			}
			return nil
		}
		if err := validateMapping(a); err != nil {
			return false, errors.Wrap(err, "left")
		}
		if err := validateMapping(b); err != nil {
			return false, errors.Wrap(err, "right")
		}

		amap := make(map[string]*yaml.Node, len(a.Content)/2)
		for i := 0; i < len(a.Content); i += 2 {
			key, val := a.Content[i], a.Content[i+1]
			if key.Kind != yaml.ScalarNode {
				return false, errors.Errorf("left: key %q is not scalar", key.Value)
			}
			amap[key.Value] = val
		}

		for i := 0; i < len(b.Content); i += 2 {
			bkey, bval := b.Content[i], b.Content[i+1]
			if bkey.Kind != yaml.ScalarNode {
				return false, errors.Errorf("right: key %q is not scalar", bkey.Value)
			}

			aval, ok := amap[bkey.Value]
			if !ok {
				return false, nil
			}
			eq, err := yamlEqual(aval, bval)
			if err != nil {
				return false, errors.Wrapf(err, "compare %q", bkey.Value)
			}
			if !eq {
				return false, nil
			}
		}
	default:
		// DocumentNode, AliasNode checked in resolveNode.
		return false, errors.Errorf("unexpected node kind: %v", a.Kind)
	}
	return true, nil
}
