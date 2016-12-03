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

var (
	wordReg        = regexp.MustCompile(`^\w+$`)
	doubleColonReg = regexp.MustCompile(`^::\w*$`)
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
			parentNode: nil,
			children:   make([]*literalNode, 0),
			handlers:   make([]*literalHandler, 0),
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
//  // node2.parentNode == node1
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
		panic(fmt.Errorf(`Multi-slash exist: "%s"`, pattern))
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
		panic(fmt.Errorf(`Path is not start with "/": "%s"`, path))
	}
	fixedLen := len(path)
	if t.fpr {
		path = fixPath(path)
		fixedLen -= len(path)
	}

	i := 0
	start := 1
	end := len(path)
	res := new(Matched)
	parent := t.root
	_path := path + "/"
	for {
		if i++; i > end {
			break
		}
		if _path[i] != '/' {
			continue
		}
		frag := _path[start:i]
		node, named := matchNode(parent, frag)
		if t.ignoreCase && node == nil {
			node, named = matchNode(parent, strings.ToLower(frag))
		}
		if node == nil {
			// TrailingSlashRedirect: /acb/efg/ -> /acb/efg
			if t.tsr && frag == "" && i == end && parent.endpoint {
				res.TSR = path[:end-1]
				if t.fpr && fixedLen > 0 {
					res.FPR = res.TSR
					res.TSR = ""
				}
			}
			return res
		}

		parent = node
		if named {
			if res.Params == nil {
				res.Params = make(map[string]string)
			}
			if parent.wildcard {
				res.Params[parent.name] = path[start:end]
				break
			} else {
				res.Params[parent.name] = frag
			}
		}
		start = i + 1
	}

	if parent.endpoint {
		res.Node = parent
		if t.fpr && fixedLen > 0 {
			res.FPR = path
			res.Node = nil
		}
	} else if t.tsr && parent.getLiteralChild("") != nil {
		// TrailingSlashRedirect: /acb/efg -> /acb/efg/
		res.TSR = path + "/"
		if t.fpr && fixedLen > 0 {
			res.FPR = res.TSR
			res.TSR = ""
		}
	}
	return res
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
	name, allow, pattern  string
	endpoint, wildcard    bool
	parentNode, varyChild *Node
	children              []*literalNode
	handlers              []*literalHandler
	regex                 *regexp.Regexp
}

type literalHandler struct {
	key string
	val interface{}
}

type literalNode struct {
	key string
	val *Node
}

func (n *Node) getLiteralChild(key string) (node *Node) {
	for _, v := range n.children {
		if key == v.key {
			node = v.val
			return
		}
	}
	return
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
	n.handlers = append(n.handlers, &literalHandler{method, handler})
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
func (n *Node) GetHandler(method string) (handler interface{}) {
	for _, v := range n.handlers {
		if method == v.key {
			handler = v.val
			return
		}
	}
	return
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
	} else if child.wildcard {
		panic(fmt.Errorf(`Can't define pattern after wildcard: "%s"`, child.pattern))
	}
	return defineNode(child, frags, ignoreCase)
}

func matchNode(parent *Node, frag string) (child *Node, named bool) {
	if child = parent.getLiteralChild(frag); child != nil {
		return
	}

	if child = parent.varyChild; child != nil {
		if child.regex != nil && !child.regex.MatchString(frag) {
			child = nil
		} else {
			named = true
		}
	}
	return
}

func parseNode(parent *Node, frag string, ignoreCase bool) *Node {
	_frag := frag
	if doubleColonReg.MatchString(frag) {
		_frag = frag[1:]
	}
	if ignoreCase {
		_frag = strings.ToLower(_frag)
	}

	if node := parent.getLiteralChild(_frag); node != nil {
		return node
	}

	node := &Node{
		parentNode: parent,
		children:   make([]*literalNode, 0),
		handlers:   make([]*literalHandler, 0),
	}

	if frag == "" {
		parent.children = append(parent.children, &literalNode{frag, node})

	} else if doubleColonReg.MatchString(frag) {
		// pattern "/a/::" should match "/a/:"
		// pattern "/a/::bc" should match "/a/:bc"
		// pattern "/a/::/bc" should match "/a/:/bc"
		parent.children = append(parent.children, &literalNode{_frag, node})
	} else if frag[0] == ':' {
		var name, regex string
		name = frag[1:]
		trailing := name[len(name)-1]
		if trailing == ')' {
			if index := strings.IndexRune(name, '('); index > 0 {
				regex = name[index+1 : len(name)-1]
				if len(regex) > 0 {
					name = name[0:index]
					node.regex = regexp.MustCompile(regex)
				} else {
					panic(fmt.Errorf(`Invalid pattern: "%s"`, frag))
				}
			}
		} else if trailing == '*' {
			name = name[0 : len(name)-1]
			node.wildcard = true
		}
		// name must be word characters `[0-9A-Za-z_]`
		if !wordReg.MatchString(name) {
			panic(fmt.Errorf(`Invalid pattern: "%s"`, frag))
		}
		node.name = name
		if child := parent.varyChild; child != nil {
			if child.name != name || child.wildcard != node.wildcard {
				panic(fmt.Errorf(`Invalid pattern: "%s"`, frag))
			}
			if child.regex != nil && child.regex.String() != regex {
				panic(fmt.Errorf(`Invalid pattern: "%s"`, frag))
			}
			return child
		}

		parent.varyChild = node
	} else if frag[0] == '*' || frag[0] == '(' || frag[0] == ')' {
		panic(fmt.Errorf(`Invalid pattern: "%s"`, frag))
	} else {
		parent.children = append(parent.children, &literalNode{_frag, node})
	}

	return node
}

func fixPath(path string) string {
	if !strings.Contains(path, "//") {
		return path
	}
	return fixPath(strings.Replace(path, "//", "/", -1))
}
