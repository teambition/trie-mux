package trie

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func EqualPtr(t *testing.T, a, b interface{}) {
	assert.Equal(t, reflect.ValueOf(a).Pointer(), reflect.ValueOf(b).Pointer())
}

func NotEqualPtr(t *testing.T, a, b interface{}) {
	assert.NotEqual(t, reflect.ValueOf(a).Pointer(), reflect.ValueOf(b).Pointer())
}

func TestGearTrieDefine(t *testing.T) {
	t.Run("root pattern", func(t *testing.T) {
		assert := assert.New(t)

		tr1 := New()
		tr2 := New()
		node := tr1.Define("/")
		assert.Equal(node.name, "")

		EqualPtr(t, node, tr1.Define("/"))
		EqualPtr(t, node, tr1.Define(""))
		NotEqualPtr(t, node, tr2.Define("/"))
		NotEqualPtr(t, node, tr2.Define(""))

		EqualPtr(t, node.parent, tr1.root)
	})

	t.Run("simple pattern", func(t *testing.T) {
		assert := assert.New(t)

		tr1 := New()
		node := tr1.Define("/a/b")
		assert.Equal(node.name, "")

		EqualPtr(t, node, tr1.Define("/a/b"))
		NotEqualPtr(t, node, tr1.Define("a/b/"))
		NotEqualPtr(t, node, tr1.Define("/a/b/"))
		EqualPtr(t, tr1.Define("/a/b/"), tr1.Define("a/b/"))
		assert.Equal(node.pattern, "/a/b")

		parent := tr1.Define("/a")
		EqualPtr(t, node.parent, parent)
		assert.Equal(0, len(parent.varyChildren))
		EqualPtr(t, parent.getChild("b"), node)
		child := tr1.Define("/a/b/c")
		EqualPtr(t, child.parent, node)
		EqualPtr(t, node.getChild("c"), child)

		assert.Panics(func() {
			tr1.Define("/a//b")
		})
	})

	t.Run("double colon pattern", func(t *testing.T) {
		assert := assert.New(t)

		tr1 := New()
		node := tr1.Define("/a/::b")
		assert.Equal(node.name, "")
		NotEqualPtr(t, node, tr1.Define("/a/::"))
		NotEqualPtr(t, node, tr1.Define("/a/::x"))

		parent := tr1.Define("/a")
		EqualPtr(t, node.parent, parent)
		assert.Equal(0, len(parent.varyChildren))
		EqualPtr(t, parent.getChild(":"), tr1.Define("/a/::"))
		EqualPtr(t, parent.getChild(":b"), tr1.Define("/a/::b"))
		EqualPtr(t, parent.getChild(":x"), tr1.Define("/a/::x"))

		child := tr1.Define("/a/::b/c")
		EqualPtr(t, child.parent, node)
		EqualPtr(t, node.getChild("c"), child)
	})

	t.Run("named pattern", func(t *testing.T) {
		assert := assert.New(t)

		tr1 := New()

		assert.Panics(func() {
			tr1.Define("/a/:")
		})
		assert.Panics(func() {
			tr1.Define("/a/:/")
		})
		assert.Panics(func() {
			tr1.Define("/a/:abc$/")
		})
		node := tr1.Define("/a/:b")
		assert.Equal(node.name, "b")
		assert.False(node.wildcard)
		assert.Equal(0, len(node.varyChildren))
		assert.Equal(node.pattern, "/a/:b")
		assert.Panics(func() {
			tr1.Define("/a/:x")
		})

		parent := tr1.Define("/a")
		assert.Equal(parent.name, "")
		EqualPtr(t, parent.varyChildren[0], node)
		EqualPtr(t, node.parent, parent)
		child := tr1.Define("/a/:b/c")
		EqualPtr(t, child.parent, node)
		assert.Panics(func() {
			tr1.Define("/a/:x/c")
		})
	})

	t.Run("named pattern with suffix", func(t *testing.T) {
		assert := assert.New(t)

		tr1 := New()
		assert.Panics(func() {
			tr1.Define("/a/:+")
		})
		assert.Panics(func() {
			tr1.Define("/a/:+a")
		})

		node1 := tr1.Define("/a/:b")
		assert.Equal(node1.name, "b")
		assert.False(node1.wildcard)
		assert.Equal(0, len(node1.varyChildren))
		assert.Equal(1, len(node1.parent.varyChildren))
		assert.Equal(node1.pattern, "/a/:b")

		parent := tr1.Define("/a")
		assert.Equal(parent.name, "")
		EqualPtr(t, parent, node1.parent)
		EqualPtr(t, parent.varyChildren[0], node1)

		node2 := tr1.Define("/a/:b+:undelete")
		assert.Equal(node2.name, "b")
		assert.False(node2.wildcard)
		assert.Equal(0, len(node2.varyChildren))
		assert.Equal(2, len(node2.parent.varyChildren))
		EqualPtr(t, tr1.Define("/a/:b+:undelete"), node2)
		EqualPtr(t, parent.varyChildren[0], node2)
		EqualPtr(t, parent.varyChildren[1], node1)

		assert.Equal(node2.pattern, "/a/:b+:undelete")

		assert.Panics(func() {
			tr1.Define("/a/:x")
		})
		assert.Panics(func() {
			tr1.Define("/a/:x+:undelete")
		})

		child := tr1.Define("/a/:b+:undelete/c")
		EqualPtr(t, child.parent, node2)
		assert.Panics(func() {
			tr1.Define("/a/:x/c")
		})
		node3 := tr1.Define("/a/:b+:delete")

		EqualPtr(t, parent.varyChildren[0], node2)
		EqualPtr(t, parent.varyChildren[1], node3)
		EqualPtr(t, parent.varyChildren[2], node1)

		tr2 := New()
		tr2.Define("/a/:b/c")
		tr2.Define("/a/:b+:delete")
		assert.Panics(func() {
			tr2.Define("/a/:x+:delete")
		})
		assert.NotPanics(func() {
			tr2.Define("/a/:b(xyz)+:delete")
		})
	})

	t.Run("wildcard pattern", func(t *testing.T) {
		assert := assert.New(t)

		tr1 := New()
		assert.Panics(func() {
			tr1.Define("/a/*")
		})
		assert.Panics(func() {
			tr1.Define("/a/:*")
		})
		assert.Panics(func() {
			tr1.Define("/a/:#*")
		})
		assert.Panics(func() {
			tr1.Define("/a/:abc(*")
		})

		node := tr1.Define("/a/:b*")
		assert.Equal(node.name, "b")
		assert.True(node.wildcard)
		assert.Equal(0, len(node.varyChildren))
		assert.Equal(node.pattern, "/a/:b*")
		assert.Panics(func() {
			tr1.Define("/a/:x*")
		})
		assert.Panics(func() {
			tr1.Define("/a/:b")
		})
		assert.Panics(func() {
			tr1.Define("/a/:b/c")
		})

		parent := tr1.Define("/a")
		assert.Equal(parent.name, "")
		assert.False(parent.wildcard)
		EqualPtr(t, parent.varyChildren[0], node)
		EqualPtr(t, node.parent, parent)
		assert.Panics(func() {
			tr1.Define("/a/:b*/c")
		})
		tr1.Define("/a/bc")
		tr1.Define("/a/b/c")
		EqualPtr(t, node, tr1.Define("/a/:b*"))
	})

	t.Run("regexp pattern", func(t *testing.T) {
		assert := assert.New(t)

		tr1 := New()
		assert.Panics(func() {
			tr1.Define("/a/(")
		})
		assert.Panics(func() {
			tr1.Define("/a/)")
		})
		assert.Panics(func() {
			tr1.Define("/a/:(")
		})
		assert.Panics(func() {
			tr1.Define("/a/:)")
		})
		assert.Panics(func() {
			tr1.Define("/a/:()")
		})
		assert.Panics(func() {
			tr1.Define("/a/:bc)")
		})
		assert.Panics(func() {
			tr1.Define("/a/:bc()")
		})
		assert.Panics(func() {
			tr1.Define("/a/:(bc)")
		})
		assert.Panics(func() {
			tr1.Define("/a/:#(bc)")
		})
		assert.Panics(func() {
			tr1.Define("/a/:b(c)*")
		})

		node := tr1.Define("/a/:b(x|y|z)")
		assert.Equal(node.name, "b")
		assert.Equal(node.pattern, "/a/:b(x|y|z)")
		assert.False(node.wildcard)
		assert.Equal(0, len(node.varyChildren))
		assert.Equal(node, tr1.Define("/a/:b(x|y|z)"))
		assert.Panics(func() {
			tr1.Define("/a/:x(x|y|z)")
		})

		NotEqualPtr(t, node, tr1.Define("/a/:b(xyz)"))

		parent := tr1.Define("/a")
		assert.Equal(parent.name, "")
		assert.False(parent.wildcard)
		EqualPtr(t, parent.varyChildren[0], node)
		EqualPtr(t, node.parent, parent)

		child := tr1.Define("/a/:b(x|y|z)/:c")
		EqualPtr(t, child.parent, node)
		assert.Panics(func() {
			tr1.Define("/a/:x(x|y|z)/:c")
		})
		assert.NotPanics(func() {
			tr1.Define("/a/:b(x|y|z)/:c(xyz)")
		})
	})

	t.Run("complex pattern", func(t *testing.T) {
		assert := assert.New(t)

		tr := New()
		p := tr.Define("/a")
		n1 := tr.Define("/a/:b")
		assert.Panics(func() {
			tr.Define("/a/:c")
		})
		n2 := tr.Define("/a/:c(x|y)")
		n3 := tr.Define("/a/:d+a1")
		n4 := tr.Define("/a/:b+a2")
		assert.Panics(func() {
			tr.Define("/a/:bb+a2")
		})
		n5 := tr.Define("/a/:b(a+)+a2")
		n6 := tr.Define("/a/:b(b+)+a2")
		n7 := tr.Define("/a/:b(c+)")
		assert.Panics(func() {
			tr.Define("/a/:bb(c+)")
		})
		n8 := tr.Define("/a/:w*")
		assert.Panics(func() {
			tr.Define("/a/:b(d+)")
		})

		EqualPtr(t, p.varyChildren[0], n5)
		EqualPtr(t, p.varyChildren[1], n6)
		EqualPtr(t, p.varyChildren[2], n3)
		EqualPtr(t, p.varyChildren[3], n4)
		EqualPtr(t, p.varyChildren[4], n2)
		EqualPtr(t, p.varyChildren[5], n7)
		EqualPtr(t, p.varyChildren[6], n1)
		EqualPtr(t, p.varyChildren[7], n8)
	})

	t.Run("ignoreCase option", func(t *testing.T) {
		tr := New(Options{IgnoreCase: true})
		node := tr.Define("/A/b")
		EqualPtr(t, node, tr.Define("/a/b"))
		EqualPtr(t, node, tr.Define("/a/B"))

		node = tr.Define("/::A/b")
		EqualPtr(t, node, tr.Define("/::a/b"))

		tr = New(Options{IgnoreCase: false})
		node = tr.Define("/A/b")
		NotEqualPtr(t, node, tr.Define("/a/b"))
		NotEqualPtr(t, node, tr.Define("/a/B"))

		node = tr.Define("/::A/b")
		NotEqualPtr(t, node, tr.Define("/::a/b"))
	})
}

func TestGearTrieMatch(t *testing.T) {
	t.Run("root pattern", func(t *testing.T) {
		assert := assert.New(t)

		tr1 := New()
		node := tr1.Define("/")
		res := tr1.Match("/")
		assert.Nil(res.Params)
		EqualPtr(t, node, res.Node)

		assert.Panics(func() {
			tr1.Match("")
		})

		assert.Nil(tr1.Match("/a").Node)
	})

	t.Run("simple pattern", func(t *testing.T) {
		assert := assert.New(t)

		tr1 := New()
		node := tr1.Define("/a/b")
		res := tr1.Match("/a/b")
		assert.Nil(res.Params)
		EqualPtr(t, node, res.Node)

		assert.Nil(tr1.Match("/a").Node)
		assert.Nil(tr1.Match("/a/b/c").Node)
		assert.Nil(tr1.Match("/a/x/c").Node)
	})

	t.Run("double colon pattern", func(t *testing.T) {
		assert := assert.New(t)

		tr1 := New()
		node := tr1.Define("/a/::b")
		res := tr1.Match("/a/:b")
		assert.Nil(res.Params)
		EqualPtr(t, node, res.Node)
		assert.Nil(tr1.Match("/a").Node)
		assert.Nil(tr1.Match("/a/::b").Node)

		node = tr1.Define("/a/::b/c")
		res = tr1.Match("/a/:b/c")
		assert.Nil(res.Params)
		EqualPtr(t, node, res.Node)
		assert.Nil(tr1.Match("/a/::b/c").Node)

		node = tr1.Define("/a/::")
		res = tr1.Match("/a/:")
		assert.Nil(res.Params)
		EqualPtr(t, node, res.Node)
		assert.Nil(tr1.Match("/a/::").Node)
	})

	t.Run("named pattern", func(t *testing.T) {
		assert := assert.New(t)

		tr1 := New()
		node := tr1.Define("/a/:b")
		res := tr1.Match("/a/xyz汉")
		assert.Equal("xyz汉", res.Params["b"])
		assert.Equal("", res.Params["x"])
		EqualPtr(t, node, res.Node)
		assert.Nil(tr1.Match("/a").Node)
		assert.Nil(tr1.Match("/a/xyz汉/123").Node)

		node2 := tr1.Define("/:a/:b")
		res2 := tr1.Match("/a/xyz汉")
		EqualPtr(t, node, res2.Node)

		res2 = tr1.Match("/ab/xyz汉")
		assert.Equal("xyz汉", res2.Params["b"])
		assert.Equal("ab", res2.Params["a"])
		EqualPtr(t, node2, res2.Node)
		assert.Nil(tr1.Match("/ab").Node)
		assert.Nil(tr1.Match("/ab/xyz汉/123").Node)
	})

	t.Run("named pattern with suffix", func(t *testing.T) {
		assert := assert.New(t)

		tr1 := New()
		node := tr1.Define("/a/:b+:del")
		res := tr1.Match("/a/xyz汉:del")
		assert.Equal("xyz汉", res.Params["b"])
		assert.Equal("", res.Params["x"])
		EqualPtr(t, node, res.Node)
		assert.Nil(tr1.Match("/a").Node)
		assert.Nil(tr1.Match("/a/:del").Node)
		assert.Nil(tr1.Match("/a/xyz汉").Node)
		assert.Nil(tr1.Match("/a/xyz汉:de").Node)
		assert.Nil(tr1.Match("/a/xyz汉/123").Node)

		node2 := tr1.Define("/a/:b+del")
		res2 := tr1.Match("/a/xyz汉del")
		assert.Equal("xyz汉", res.Params["b"])
		EqualPtr(t, node2, res2.Node)
		assert.Nil(tr1.Match("/a/xyz汉cel").Node)
	})

	t.Run("wildcard pattern", func(t *testing.T) {
		assert := assert.New(t)

		tr1 := New()
		node := tr1.Define("/a/:b*")
		res := tr1.Match("/a/xyz汉")
		assert.Equal("xyz汉", res.Params["b"])
		EqualPtr(t, node, res.Node)
		assert.Nil(tr1.Match("/a").Node)

		res = tr1.Match("/a/xyz汉/123")
		assert.Equal("xyz汉/123", res.Params["b"])
		EqualPtr(t, node, res.Node)

		node = tr1.Define("/:a*")
		assert.Nil(tr1.Match("/a").Node) // TODO
		res = tr1.Match("/123")
		assert.Equal("123", res.Params["a"])
		EqualPtr(t, node, res.Node)
		res = tr1.Match("/123/xyz汉")
		assert.Equal("123/xyz汉", res.Params["a"])
		EqualPtr(t, node, res.Node)
	})

	t.Run("regexp pattern", func(t *testing.T) {
		assert := assert.New(t)

		tr1 := New()
		node := tr1.Define("/a/:b(^(x|y|z)$)")
		res := tr1.Match("/a/x")
		assert.Equal("x", res.Params["b"])
		EqualPtr(t, node, res.Node)
		res = tr1.Match("/a/y")
		assert.Equal("y", res.Params["b"])
		EqualPtr(t, node, res.Node)
		res = tr1.Match("/a/z")
		assert.Equal("z", res.Params["b"])
		EqualPtr(t, node, res.Node)

		assert.Nil(tr1.Match("/a").Node)
		assert.Nil(tr1.Match("/a/xy").Node)
		assert.Nil(tr1.Match("/a/x/y").Node)

		child := tr1.Define("/a/:b(^(x|y|z)$)/c")
		res = tr1.Match("/a/x/c")
		assert.Equal("x", res.Params["b"])
		EqualPtr(t, child, res.Node)
		res = tr1.Match("/a/y/c")
		assert.Equal("y", res.Params["b"])
		EqualPtr(t, child, res.Node)
		res = tr1.Match("/a/z/c")
		assert.Equal("z", res.Params["b"])
		EqualPtr(t, child, res.Node)
	})

	t.Run("complex regexp pattern", func(t *testing.T) {
		assert := assert.New(t)

		tr := New()
		node1 := tr.Define("/thumbnail/:fileKey/w/:width/h/:height")
		node2 := tr.Define("/thumbnail/:h1(^\\w{2}$)/:h2/:h3/w/:width/h/:height")

		res := tr.Match("/thumbnail/50c32afae8cf1439d35a87e6/w/200/h/200")
		EqualPtr(t, node1, res.Node)
		assert.Equal("/thumbnail/:fileKey/w/:width/h/:height", res.Node.GetPattern())
		assert.Equal("/thumbnail/:h1(^\\w{2}$)/:h2/:h3/w/:width/h/:height", node2.GetPattern())
		assert.Equal("50c32afae8cf1439d35a87e6", res.Params["fileKey"])
		assert.Equal("200", res.Params["width"])
		assert.Equal("200", res.Params["height"])

		res = tr.Match("/thumbnail/50/c3/2afae8cf1439d35a87e6/w/200/h/200")
		EqualPtr(t, node2, res.Node)
		assert.Equal("50", res.Params["h1"])
		assert.Equal("c3", res.Params["h2"])
		assert.Equal("2afae8cf1439d35a87e6", res.Params["h3"])
		assert.Equal("200", res.Params["width"])
		assert.Equal("200", res.Params["height"])
	})

	t.Run("regexp pattern with suffix", func(t *testing.T) {
		assert := assert.New(t)

		tr1 := New()
		node := tr1.Define("/a/:b(^(x|y)$)+:cancel")
		assert.Nil(tr1.Match("/a/x").Node)
		res := tr1.Match("/a/x:cancel")
		assert.Equal("x", res.Params["b"])
		EqualPtr(t, node, res.Node)
		res = tr1.Match("/a/y:cancel")
		assert.Equal("y", res.Params["b"])
		EqualPtr(t, node, res.Node)
		assert.Nil(tr1.Match("/a/z:cancel").Node)

		node = tr1.Define("/a/:b(^(x|y)$)++undelete")
		assert.Nil(tr1.Match("/a/x").Node)
		res = tr1.Match("/a/x+undelete")
		assert.Equal("x", res.Params["b"])
		EqualPtr(t, node, res.Node)
		res = tr1.Match("/a/y+undelete")
		assert.Equal("y", res.Params["b"])
		EqualPtr(t, node, res.Node)
		assert.Nil(tr1.Match("/a/z+undelete").Node)

		node = tr1.Define("/a/:b(^(a|z)$)++undelete")
		assert.Nil(tr1.Match("/a/x").Node)
		res = tr1.Match("/a/a+undelete")
		assert.Equal("a", res.Params["b"])
		EqualPtr(t, node, res.Node)
		res = tr1.Match("/a/z+undelete")
		assert.Equal("z", res.Params["b"])
		EqualPtr(t, node, res.Node)
		res = tr1.Match("/a/y+undelete")
		assert.Equal("y", res.Params["b"])
		EqualPtr(t, tr1.Define("/a/:b(^(x|y)$)++undelete"), res.Node)

		node = tr1.Define(`/api/:resource/:ID(^\d+$)+:cancel`)
		res = tr1.Match("/api/task/123:cancel")
		assert.Equal("task", res.Params["resource"])
		assert.Equal("123", res.Params["ID"])
		EqualPtr(t, node, res.Node)
	})

	t.Run("IgnoreCase option", func(t *testing.T) {
		assert := assert.New(t)

		// IgnoreCase = true
		tr := New(Options{IgnoreCase: true})
		node := tr.Define("/A/:Name")

		res := tr.Match("/a/x")
		EqualPtr(t, node, res.Node)
		assert.Equal("x", res.Params["Name"])
		assert.Equal("", res.Params["name"])

		res = tr.Match("/A/X")
		EqualPtr(t, node, res.Node)
		assert.Equal("X", res.Params["Name"])
		assert.Equal("", res.Params["name"])

		node = tr.Define("/::A/:Name")

		res = tr.Match("/:a/x")
		EqualPtr(t, node, res.Node)
		assert.Equal("x", res.Params["Name"])
		assert.Equal("", res.Params["name"])

		res = tr.Match("/:A/X")
		EqualPtr(t, node, res.Node)
		assert.Equal("X", res.Params["Name"])
		assert.Equal("", res.Params["name"])

		// IgnoreCase = false
		tr = New(Options{IgnoreCase: false})
		node = tr.Define("/A/:Name")

		assert.Nil(tr.Match("/a/x").Node)
		res = tr.Match("/A/X")
		EqualPtr(t, node, res.Node)
		assert.Equal("X", res.Params["Name"])

		node = tr.Define("/::A/:Name")

		assert.Nil(tr.Match("/:a/x").Node)
		res = tr.Match("/:A/X")
		EqualPtr(t, node, res.Node)
		assert.Equal("X", res.Params["Name"])
		assert.Equal("", res.Params["name"])

		tr = New(Options{IgnoreCase: true})
		node1 := tr.Define("/tasks/:taskId")
		node2 := tr.Define("/tasks/batchGet")

		res = tr.Match("/tasks/batchGet")
		EqualPtr(t, node2, res.Node)

		res = tr.Match("/tasks/batchget")
		EqualPtr(t, node2, res.Node)

		tr = New(Options{IgnoreCase: false})
		node1 = tr.Define("/tasks/:taskId")
		node2 = tr.Define("/tasks/batchGet")

		res = tr.Match("/tasks/batchGet")
		EqualPtr(t, node2, res.Node)

		res = tr.Match("/tasks/batchget")
		EqualPtr(t, node1, res.Node)
		assert.Equal("batchget", res.Params["taskId"])
	})

	t.Run("FixedPathRedirect option", func(t *testing.T) {
		assert := assert.New(t)

		// FixedPathRedirect = false
		tr := New(Options{FixedPathRedirect: false})
		node1 := tr.Define("/abc/efg")
		node2 := tr.Define("/abc/xyz/")

		EqualPtr(t, node1, tr.Match("/abc/efg").Node)
		assert.Equal("", tr.Match("/abc/efg").FPR)
		assert.Nil(tr.Match("/abc//efg").Node)
		assert.Equal("", tr.Match("/abc//efg").FPR)

		EqualPtr(t, node2, tr.Match("/abc/xyz/").Node)
		assert.Equal("", tr.Match("/abc/xyz/").FPR)
		assert.Nil(tr.Match("/abc/xyz//").Node)
		assert.Equal("", tr.Match("/abc/xyz//").FPR)

		// FixedPathRedirect = true
		tr = New(Options{FixedPathRedirect: true})
		node1 = tr.Define("/abc/efg")
		node2 = tr.Define("/abc/xyz/")

		EqualPtr(t, node1, tr.Match("/abc/efg").Node)
		assert.Equal("", tr.Match("/abc/efg").FPR)
		assert.Nil(tr.Match("/abc//efg").Node)
		assert.Equal("/abc/efg", tr.Match("/abc//efg").FPR)
		assert.Nil(tr.Match("/abc///efg").Node)
		assert.Equal("/abc/efg", tr.Match("/abc///efg").FPR)

		EqualPtr(t, node2, tr.Match("/abc/xyz/").Node)
		assert.Equal("", tr.Match("/abc/xyz/").FPR)
		assert.Nil(tr.Match("/abc/xyz//").Node)
		assert.Equal("/abc/xyz/", tr.Match("/abc/xyz//").FPR)
		assert.Nil(tr.Match("/abc/xyz////").Node)
		assert.Equal("/abc/xyz/", tr.Match("/abc/xyz////").FPR)
	})

	t.Run("TrailingSlashRedirect option", func(t *testing.T) {
		assert := assert.New(t)

		// TrailingSlashRedirect = false
		tr := New(Options{TrailingSlashRedirect: false})
		node1 := tr.Define("/abc/efg")
		node2 := tr.Define("/abc/xyz/")

		EqualPtr(t, node1, tr.Match("/abc/efg").Node)
		assert.Equal("", tr.Match("/abc/efg").TSR)
		assert.Nil(tr.Match("/abc/efg/").Node)
		assert.Equal("", tr.Match("/abc/efg/").TSR)

		EqualPtr(t, node2, tr.Match("/abc/xyz/").Node)
		assert.Equal("", tr.Match("/abc/xyz/").TSR)
		assert.Nil(tr.Match("/abc/xyz").Node)
		assert.Equal("", tr.Match("/abc/xyz").TSR)

		// TrailingSlashRedirect = true
		tr = New(Options{TrailingSlashRedirect: true})
		node1 = tr.Define("/abc/efg")
		node2 = tr.Define("/abc/xyz/")

		EqualPtr(t, node1, tr.Match("/abc/efg").Node)
		assert.Equal("", tr.Match("/abc/efg").TSR)
		assert.Nil(tr.Match("/abc/efg/").Node)
		assert.Equal("/abc/efg", tr.Match("/abc/efg/").TSR)

		EqualPtr(t, node2, tr.Match("/abc/xyz/").Node)
		assert.Equal("", tr.Match("/abc/xyz/").TSR)
		assert.Nil(tr.Match("/abc/xyz").Node)
		assert.Equal("/abc/xyz/", tr.Match("/abc/xyz").TSR)

		// TrailingSlashRedirect = true and FixedPathRedirect = true
		tr = New(Options{FixedPathRedirect: true, TrailingSlashRedirect: true})
		node1 = tr.Define("/abc/efg")
		node2 = tr.Define("/abc/xyz/")

		assert.Nil(tr.Match("/abc//efg/").Node)
		assert.Equal("", tr.Match("/abc//efg/").TSR)
		assert.Equal("/abc/efg", tr.Match("/abc//efg/").FPR)

		assert.Nil(tr.Match("/abc//xyz").Node)
		assert.Equal("", tr.Match("/abc//xyz").TSR)
		assert.Equal("/abc/xyz/", tr.Match("/abc//xyz").FPR)

		tr = New(Options{TrailingSlashRedirect: false})
		node1 = tr.Define("/abc/:name")
		node2 = tr.Define("/abc")

		EqualPtr(t, node1, tr.Match("/abc/efg").Node)
		assert.Equal("", tr.Match("/abc/efg").TSR)
		assert.Nil(tr.Match("/abc/").Node)
		assert.Equal("", tr.Match("/abc/").TSR)

		tr = New(Options{TrailingSlashRedirect: true})
		node1 = tr.Define("/abc/:name")
		node2 = tr.Define("/abc")

		EqualPtr(t, node1, tr.Match("/abc/efg").Node)
		assert.Equal("", tr.Match("/abc/efg").TSR)
		assert.Nil(tr.Match("/abc/").Node)
		assert.Equal("/abc", tr.Match("/abc/").TSR)
	})
}

func TestGearTrieNode(t *testing.T) {
	t.Run("Node Handle", func(t *testing.T) {
		assert := assert.New(t)

		handler := func() {}
		tr := New()
		tr.Define("/").Handle("GET", handler)
		tr.Define("/").Handle("PUT", handler)
		tr.Define("/api?pageSize=&pageToken=").Handle("GET", handler)

		assert.Panics(func() {
			tr.Define("/").Handle("GET", handler)
		})
		assert.Panics(func() {
			tr.Define("/").Handle("PUT", handler)
		})
		assert.Panics(func() {
			tr.Define("/api").Handle("GET", handler)
		})

		EqualPtr(t, handler, tr.Match("/").Node.GetHandler("GET").(func()))
		EqualPtr(t, handler, tr.Match("/").Node.GetHandler("PUT").(func()))
		assert.Equal("GET, PUT", tr.Match("/").Node.GetAllow())

		EqualPtr(t, handler, tr.Match("/api").Node.GetHandler("GET").(func()))
		assert.Equal("GET", tr.Match("/api").Node.GetAllow())

		for _, node := range tr.GetEndpoints() {
			fmt.Println(node.GetMethods(), node.GetPattern())
		}
	})
}
