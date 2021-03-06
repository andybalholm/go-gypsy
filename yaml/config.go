package yaml

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// A File represents the top-level YAML node found in a file.  It is intended
// for use as a configuration file.
type File struct {
	Root Node

	// TODO(kevlar): Add a cache?
}

// ReadFile reads a YAML configuration file from the given filename.
func ReadFile(filename string) (*File, os.Error) {
	fin, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	f := new(File)
	f.Root, err = Parse(fin)
	if err != nil {
		return nil, err
	}

	return f, nil
}

// Config reads a YAML configuration from a static string.  If an error is
// found, it will panic.  This is a utility function and is intended for use in
// initializers.
func Config(yamlconf string) *File {
	var err os.Error
	buf := bytes.NewBufferString(yamlconf)

	f := new(File)
	f.Root, err = Parse(buf)
	if err != nil {
		panic(err)
	}

	return f
}

// ConfigFile reads a YAML configuration file from the given filename and
// panics if an error is found.  This is a utility function and is intended for
// use in initializers.
func ConfigFile(filename string) *File {
	f, err := ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return f
}

// Get retrieves a scalar from the file specified by a string of the same
// format as that expected by Child.  If the final node is not a Scalar, Get
// will return an error.
func (f *File) Get(spec string) (string, os.Error) {
	node, err := Child(f.Root, spec)
	if err != nil {
		return "", err
	}

	scalar, ok := node.(Scalar)
	if !ok {
		return "", &NodeTypeMismatch{
			Full:     spec,
			Spec:     spec,
			Token:    "$",
			Expected: "yaml.Scalar",
			Node:     node,
		}
	}
	return scalar.String(), nil
}

// Require retrieves a scalar from the file specified by a string of the same
// format as that expected by Child.  If the final node is not a Scalar, String
// will panic.  This is a convenience function for use in initializers.
func (f *File) Require(spec string) string {
	str, err := f.Get(spec)
	if err != nil {
		panic(err)
	}
	return str
}

// Child retrieves a child node from the specified node as follows:
//   .mapkey   - Get the key 'mapkey' of the Node, which must be a Map
//   [idx]     - Choose the index from the current Node, which must be a List
//
// The above selectors may be applied recursively, and each successive selector
// applies to the result of the previous selector.  For convenience, a "." is
// implied as the first character if the first character is not a "." or "[".
// The node tree is walked from the given node, considering each token of the
// above format.  If a node along the evaluation path is not found, an error is
// returned. If a node is not the proper type, an error is returned.  If the
// final node is not a Scalar, an error is returned. 
func Child(root Node, spec string) (Node, os.Error) {
	if len(spec) == 0 {
		return root, nil
	}

	if first := spec[0]; first != '.' && first != '[' {
		spec = "." + spec
	}

	var recur func(Node, string, string) (Node, os.Error)
	recur = func(n Node, last, s string) (Node, os.Error) {

		if len(s) == 0 {
			return n, nil
		}

		if n == nil {
			return nil, &NodeNotFound{
				Full: spec,
				Spec: last,
			}
		}

		// Extract the next token
		delim := 1 + strings.IndexAny(s[1:], ".[")
		if delim <= 0 {
			delim = len(s)
		}
		tok := s[:delim]
		remain := s[delim:]

		switch s[0] {
		case '[':
			s, ok := n.(List)
			if !ok {
				return nil, &NodeTypeMismatch{
					Node:     n,
					Expected: "yaml.List",
					Full:     spec,
					Spec:     last,
					Token:    tok,
				}
			}

			if tok[0] == '[' && tok[len(tok)-1] == ']' {
				if num, err := strconv.Atoi(tok[1 : len(tok)-1]); err == nil {
					if num >= 0 && num < len(s) {
						return recur(s[num], last+tok, remain)
					}
				}
			}
			return nil, &NodeNotFound{
				Full: spec,
				Spec: last + tok,
			}
		default:
			m, ok := n.(Map)
			if !ok {
				return nil, &NodeTypeMismatch{
					Node:     n,
					Expected: "yaml.Map",
					Full:     spec,
					Spec:     last,
					Token:    tok,
				}
			}

			n, ok = m[tok[1:]]
			return recur(n, last+tok, remain)
		}
		panic("unreachable")
	}
	return recur(root, "", spec)
}

type NodeNotFound struct {
	Full string
	Spec string
}

func (e *NodeNotFound) String() string {
	return fmt.Sprintf("yaml: %s: %q not found", e.Full, e.Spec)
}

type NodeTypeMismatch struct {
	Full     string
	Spec     string
	Token    string
	Node     Node
	Expected string
}

func (e *NodeTypeMismatch) String() string {
	return fmt.Sprintf("yaml: %s: type mismatch: %q is %T, want %s (at %q)",
		e.Full, e.Spec, e.Node, e.Expected, e.Token)
}
