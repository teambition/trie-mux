// Package trie implements a minimal and powerful trie based url path router (or mux) for Go.
package trie

import (
	"fmt"
	"regexp"
	"strings"
)

// Options is options for Trie.
type Options struct {
	// Ignore case when matching URL path.
	IgnoreCase bool

	// If enabled, the trie will detect if the current path can't be matched but
	// a handler for the fixed path exists.
	// Matched.FPR will returns either a fixed redirect path or an empty string.
	// For example when "/api/foo" defined and matching "/api//foo",
	// The result Matched.FPR is "/api/foo".
	FixedPathRedirect bool

	// If enabled, the trie will detect if the current path can't be matched but
	// a handler for the path with (without) the trailing slash exists.
	// Matched.TSR will returns either a redirect path or an empty string.
	// For example if /foo/ is requested but a route only exists for /foo, the
	// client is redirected to /foo
	// For example when "/api/foo" defined and matching "/api/foo/",
	// The result Matched.TSR is "/api/foo".
	TrailingSlashRedirect bool
}

// the valid characters for the path component are:
// [A-Za-z0-9!$%&'()*+,-.:;=@_~]
// http://stackoverflow.com/questions/4669692/valid-characters-for-directory-part-of-a-url-for-short-links

var (
	wordReg        = regexp.MustCompile(`^\w+$`)
	suffixReg      = regexp.MustCompile(`\+[A-Za-z0-9!$%&'*+,-.:;=@_~]*$`)
	doubleColonReg = regexp.MustCompile(`^::[A-Za-z0-9!$%&'*+,-.:;=@_~]*$`)
	multiSlashReg  = regexp.MustCompile(`/{2,}`)
	defaultOptions = Options{
		IgnoreCase:            true,
		TrailingSlashRedirect: true,
		FixedPathRedirect:     true,
	}
)

// New returns a trie
//
//  trie := New()
//  // disable IgnoreCase, TrailingSlashRedirect and FixedPathRedirect
//  trie := New(Options{})
//
func New(args ...Options) *Trie {
	opts := defaultOptions
	if len(args) > 0 {
		opts = args[0]
	}

	return &Trie{
		ignoreCase: opts.IgnoreCase,
		fpr:        opts.FixedPathRedirect,
		tsr:        opts.TrailingSlashRedirect,
		root: &Node{
			parent:   nil,
			children: make(map[string]*Node),
			handlers: make(map[string]interface{}),
		},
	}
}

// Trie represents a trie that defining patterns and matching URL.
type Trie struct {
	ignoreCase bool
	fpr        bool
	tsr        bool
	root       *Node
}

// Define define a pattern on the trie and returns the endpoint node for the pattern.
//
//  trie := New()
//  node1 := trie.Define("/a")
//  node2 := trie.Define("/a/b")
//  node3 := trie.Define("/a/b")
//  // node2.parent == node1
//  // node2 == node3
//
// The defined pattern can contain three types of parameters:
//
// | Syntax | Description |
// |--------|------|
// | `:name` | named parameter |
// | `:name*` | named with catch-all parameter |
// | `:name(regexp)` | named with regexp parameter |
// | `::name` | not named parameter, it is literal `:name` |
//
func (t *Trie) Define(pattern string) *Node {
	if strings.Contains(pattern, "//") {
		panic(fmt.Errorf(`multi-slash exist: "%s"`, pattern))
	}

	_pattern := strings.TrimPrefix(pattern, "/")
	node := defineNode(t.root, strings.Split(_pattern, "/"), t.ignoreCase)

	if node.pattern == "" {
		node.pattern = pattern
	}
	return node
}

// Match try to match path. It will returns a Matched instance that
// includes	*Node, Params and Tsr flag when matching success, otherwise a nil.
//
//  matched := trie.Match("/a/b")
//
func (t *Trie) Match(path string) *Matched {
	if path == "" || path[0] != '/' {
		panic(fmt.Errorf(`path is not start with "/": "%s"`, path))
	}
	fixedLen := len(path)
	if t.fpr {
		path = fixPath(path)
		fixedLen -= len(path)
	}

	start := 1
	end := len(path)
	matched := new(Matched)
	parent := t.root
	for i := 1; i <= end; i++ {
		if i < end && path[i] != '/' {
			continue
		}
		frag := path[start:i]
		node := matchNode(parent, frag)
		if t.ignoreCase && node == nil {
			node = matchNode(parent, strings.ToLower(frag))
		}
		if node == nil {
			// TrailingSlashRedirect: /acb/efg/ -> /acb/efg
			if t.tsr && parent.endpoint && i == end && frag == "" {
				matched.TSR = path[:end-1]
				if t.fpr && fixedLen > 0 {
					matched.FPR = matched.TSR
					matched.TSR = ""
				}
			}
			return matched
		}

		parent = node
		if parent.name != "" {
			if matched.Params == nil {
				matched.Params = make(map[string]string)
			}
			if parent.wildcard {
				matched.Params[parent.name] = path[start:end]
				break
			} else {
				if parent.suffix != "" {
					frag = frag[0 : len(frag)-len(parent.suffix)]
				}
				matched.Params[parent.name] = frag
			}
		}
		start = i + 1
	}

	switch {
	case parent.endpoint:
		matched.Node = parent
		if t.fpr && fixedLen > 0 {
			matched.FPR = path
			matched.Node = nil
		}
	case t.tsr && parent.getChild("") != nil:
		// TrailingSlashRedirect: /acb/efg -> /acb/efg/
		matched.TSR = path + "/"
		if t.fpr && fixedLen > 0 {
			matched.FPR = matched.TSR
			matched.TSR = ""
		}
	}

	return matched
}

// Matched is a result returned by Trie.Match.
type Matched struct {
	// Either a Node pointer when matched or nil
	Node *Node

	// Either a map contained matched values or empty map.
	Params map[string]string

	// If FixedPathRedirect enabled, it may returns a redirect path,
	// otherwise a empty string.
	FPR string

	// If TrailingSlashRedirect enabled, it may returns a redirect path,
	// otherwise a empty string.
	TSR string
}

// Node represents a node on defined patterns that can be matched.
type Node struct {
	name, allow, pattern, frag, suffix string
	endpoint, wildcard                 bool
	parent                             *Node
	varyChildren                       []*Node
	children                           map[string]*Node
	handlers                           map[string]interface{}
	regex                              *regexp.Regexp
}

func (n *Node) getFrags() string {
	frags := n.frag
	if n.parent != nil {
		frags = n.parent.getFrags() + "/" + frags
	}
	return frags
}

func (n *Node) getChild(key string) *Node {
	return n.children[key]
}

// Handle is used to mount a handler with a method name to the node.
//
//  t := New()
//  node := t.Define("/a/b")
//  node.Handle("GET", handler1)
//  node.Handle("POST", handler1)
//
func (n *Node) Handle(method string, handler interface{}) {
	if n.GetHandler(method) != nil {
		panic(fmt.Errorf(`"%s" already defined`, n.pattern))
	}
	n.handlers[method] = handler
	if n.allow == "" {
		n.allow = method
	} else {
		n.allow += ", " + method
	}
}

// GetHandler ...
// GetHandler returns handler by method that defined on the node
//
//  trie := New()
//  trie.Define("/api").Handle("GET", func handler1() {})
//  trie.Define("/api").Handle("PUT", func handler2() {})
//
//  trie.Match("/api").Node.GetHandler("GET").(func()) == handler1
//  trie.Match("/api").Node.GetHandler("PUT").(func()) == handler2
//
func (n *Node) GetHandler(method string) interface{} {
	return n.handlers[method]
}

// GetAllow returns allow methods defined on the node
//
//  trie := New()
//  trie.Define("/").Handle("GET", handler1)
//  trie.Define("/").Handle("PUT", handler2)
//
//  // trie.Match("/").Node.GetAllow() == "GET, PUT"
//
func (n *Node) GetAllow() string {
	return n.allow
}

func defineNode(parent *Node, frags []string, ignoreCase bool) *Node {
	frag := frags[0]
	frags = frags[1:]
	child := parseNode(parent, frag, ignoreCase)

	if len(frags) == 0 {
		child.endpoint = true
		return child
	}
	if child.wildcard {
		panic(fmt.Errorf(`can't define pattern after wildcard: "%s"`, child.getFrags()))
	}
	return defineNode(child, frags, ignoreCase)
}

func matchNode(parent *Node, frag string) (child *Node) {
	if child = parent.getChild(frag); child != nil {
		return
	}
	for _, child = range parent.varyChildren {
		_frag := frag
		if child.suffix != "" {
			if frag == child.suffix || !strings.HasSuffix(frag, child.suffix) {
				continue
			}
			_frag = frag[0 : len(frag)-len(child.suffix)]
		}
		if child.regex != nil && !child.regex.MatchString(_frag) {
			continue
		}
		return
	}
	return nil
}

func parseNode(parent *Node, frag string, ignoreCase bool) *Node {
	_frag := frag
	if doubleColonReg.MatchString(frag) {
		_frag = frag[1:]
	}
	if ignoreCase {
		_frag = strings.ToLower(_frag)
	}
	if node := parent.getChild(_frag); node != nil {
		return node
	}

	node := &Node{
		frag:     frag,
		parent:   parent,
		children: make(map[string]*Node),
		handlers: make(map[string]interface{}),
	}

	switch {
	case frag == "":
		parent.children[frag] = node

	case doubleColonReg.MatchString(frag):
		// pattern "/a/::" should match "/a/:"
		// pattern "/a/::bc" should match "/a/:bc"
		// pattern "/a/::/bc" should match "/a/:/bc"
		parent.children[_frag] = node

	case frag[0] == ':':
		var name, regex, suffix string
		name = frag[1:]

		switch name[len(name)-1] {
		case '*':
			name = name[0 : len(name)-1]
			node.wildcard = true

		default:
			suffix = suffixReg.FindString(name)
			if suffix != "" {
				name = name[0 : len(name)-len(suffix)]
				node.suffix = suffix[1:]
				if node.suffix == "" {
					panic(fmt.Errorf(`invalid pattern: "%s"`, frag))
				}
			}

			if name[len(name)-1] == ')' {
				if index := strings.IndexRune(name, '('); index > 0 {
					regex = name[index+1 : len(name)-1]
					if len(regex) > 0 {
						name = name[0:index]
						node.regex = regexp.MustCompile(regex)
					} else {
						panic(fmt.Errorf(`invalid pattern: "%s"`, frag))
					}
				}
			}
		}

		// name must be word characters `[0-9A-Za-z_]`
		if !wordReg.MatchString(name) {
			panic(fmt.Errorf(`invalid pattern: "%s"`, frag))
		}
		node.name = name
		// check if node exists
		for _, child := range parent.varyChildren {
			if child.name != node.name {
				panic(fmt.Errorf(`invalid pattern: "%s"`, frag))
			}
			if child.wildcard {
				if !node.wildcard {
					panic(fmt.Errorf(`can't define "%s" after: "%s"`, node.getFrags(), child.getFrags()))
				}
				return child
			}
			if child.suffix == "" && child.regex == nil && (node.suffix != "" || node.regex != nil) {
				panic(fmt.Errorf(`can't define "%s" after: "%s"`, node.getFrags(), child.getFrags()))
			}
			if child.suffix == node.suffix {
				if child.regex == nil && node.regex == nil {
					return child
				}
				if child.regex != nil && node.regex != nil && child.regex.String() == node.regex.String() {
					return child
				}
				if child.regex == nil && node.regex != nil {
					panic(fmt.Errorf(`invalid pattern: "%s"`, frag))
				}
			}
		}
		parent.varyChildren = append(parent.varyChildren, node)

	case frag[0] == '*' || frag[0] == '(' || frag[0] == ')':
		panic(fmt.Errorf(`invalid pattern: "%s"`, frag))

	default:
		parent.children[_frag] = node
	}

	return node
}

func fixPath(path string) string {
	if !strings.Contains(path, "//") {
		return path
	}
	return multiSlashReg.ReplaceAllString(path, "/")
}
