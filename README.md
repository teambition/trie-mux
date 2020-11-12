# trie-mux

A minimal and powerful trie based url path router (or mux) for Go.

[![Build Status](http://img.shields.io/travis/teambition/trie-mux.svg?style=flat-square)](https://travis-ci.org/teambition/trie-mux)
[![Coverage Status](http://img.shields.io/coveralls/teambition/trie-mux.svg?style=flat-square)](https://coveralls.io/r/teambition/trie-mux)
[![License](http://img.shields.io/badge/license-mit-blue.svg?style=flat-square)](https://raw.githubusercontent.com/teambition/trie-mux/master/LICENSE)
[![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](http://godoc.org/github.com/teambition/trie-mux)

## JavaScript Version

https://github.com/zensh/route-trie

## Features

1. Support named parameter (package trie)
1. Support regexp (package trie)
1. Support suffix matching (package trie)
1. Fixed path automatic redirection (package trie)
1. Trailing slash automatic redirection (package trie)
1. Automatic handle `405 Method Not Allowed` (package mux)
1. Automatic handle `501 Not Implemented` (package mux)
1. Automatic handle `OPTIONS` method (package mux)
1. Best Performance

## Implementations

### trie-mux: mux.Mux

https://github.com/teambition/trie-mux/blob/master/mux/mux.go

```go
package main

import (
  "fmt"
  "io/ioutil"
  "net/http"
  "net/http/httptest"

  "github.com/teambition/trie-mux/mux"
)

func main() {
  router := mux.New()
  router.Get("/", func(w http.ResponseWriter, _ *http.Request, _ mux.Params) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.WriteHeader(200)
    w.Write([]byte("<h1>Hello, Gear!</h1>"))
  })

  router.Get("/view/:view", func(w http.ResponseWriter, _ *http.Request, params mux.Params) {
    view := params["view"]
    if view == "" {
      http.Error(w, "Invalid view", 400)
    } else {
      w.Header().Set("Content-Type", "text/html; charset=utf-8")
      w.WriteHeader(200)
      w.Write([]byte("View: " + view))
    }
  })

  // srv := http.Server{Addr: ":3000", Handler: router}
  // srv.ListenAndServe()
  srv := httptest.NewServer(router)
  defer srv.Close()

  res, _ := http.Get(srv.URL + "/view/users")
  body, _ := ioutil.ReadAll(res.Body)
  res.Body.Close()

  fmt.Println(res.StatusCode, string(body))
  // Output: 200 View: users
}
```

### Gear: gear.Router

https://github.com/teambition/gear/blob/master/router.go

```go
package main

import (
  "fmt"
  "io/ioutil"
  "net/http"

  "github.com/teambition/gear"
)

func main() {
  app := gear.New()

  router := gear.NewRouter()
  router.Get("/", func(ctx *gear.Context) error {
    return ctx.HTML(200, "<h1>Hello, Gear!</h1>")
  })
  router.Get("/view/:view", func(ctx *gear.Context) error {
    view := ctx.Param("view")
    if view == "" {
      return &gear.Error{Code: 400, Msg: "Invalid view"}
    }
    return ctx.HTML(200, "View: "+view)
  })

  app.UseHandler(router)
  srv := app.Start(":3000")
  defer srv.Close()

  res, _ := http.Get("http://" + srv.Addr().String() + "/view/users")
  body, _ := ioutil.ReadAll(res.Body)
  res.Body.Close()

  fmt.Println(res.StatusCode, string(body))
  // Output: 200 View: users
}
```

## Pattern Rule

The defined pattern can contain six types of parameters:

| Syntax | Description |
|--------|------|
| `:name` | named parameter |
| `:name(regexp)` | named with regexp parameter |
| `:name+suffix` | named parameter with suffix matching |
| `:name(regexp)+suffix` | named with regexp parameter and suffix matching |
| `:name*` | named with catch-all parameter |
| `::name` | not named parameter, it is literal `:name` |

Named parameters are dynamic path segments. They match anything until the next '/' or the path end:

Defined: `/api/:type/:ID`
```
/api/user/123             matched: type="user", ID="123"
/api/user                 no match
/api/user/123/comments    no match
```

Named with regexp parameters match anything using regexp until the next '/' or the path end:

Defined: `/api/:type/:ID(^\d+$)`
```
/api/user/123             matched: type="user", ID="123"
/api/user                 no match
/api/user/abc             no match
/api/user/123/comments    no match
```

Named parameters with suffix, such as [Google API Design](https://cloud.google.com/apis/design/custom_methods):

Defined: `/api/:resource/:ID+:undelete`
```
/api/file/123                     no match
/api/file/123:undelete            matched: resource="file", ID="123"
/api/file/123:undelete/comments   no match
```

Named with regexp parameters and suffix:

Defined: `/api/:resource/:ID(^\d+$)+:cancel`
```
/api/task/123                   no match
/api/task/123:cancel            matched: resource="task", ID="123"
/api/task/abc:cancel            no match
```

Named with catch-all parameters match anything until the path end, including the directory index (the '/' before the catch-all). Since they match anything until the end, catch-all parameters must always be the final path element.

Defined: `/files/:filepath*`
```
/files                           no match
/files/LICENSE                   matched: filepath="LICENSE"
/files/templates/article.html    matched: filepath="templates/article.html"
```

The value of parameters is saved on the `Matched.Params`. Retrieve the value of a parameter by name:
```
type := matched.Params("type")
id   := matched.Params("ID")
```

Url query string with `?` can be provided when defining trie, but it will be ignored.

Defined: `/files?pageSize=&pageToken=`
Equal to: `/files`
```
/files                           matched, query string will be ignored
/files/LICENSE                   no match
```

## Documentation

https://godoc.org/github.com/teambition/trie-mux

## Bench

```bash
go test -bench=. ./mux
```

```
GithubAPI Routes: 203
   trie-mux: 37464 Bytes
   HttpRouter: 37464 Bytes
   httptreemux: 78768 Bytes
BenchmarkTrieMux-4                    20000      758711 ns/op   1082902 B/op      2974 allocs/op
BenchmarkHttpRouter-4                 20000      687400 ns/op   1030826 B/op      2604 allocs/op
BenchmarkHttpTreeMux-4                20000      786506 ns/op   1082902 B/op      3108 allocs/op
BenchmarkTrieMuxRequests-4             1000    17564036 ns/op    816398 B/op     10488 allocs/op
BenchmarkHttpRouterRequests-4          1000    17050872 ns/op    764200 B/op     10117 allocs/op
BenchmarkHttpTreeMuxRequests-4         1000    17125625 ns/op    816408 B/op     10622 allocs/op
PASS
ok    github.com/teambition/trie-mux/mux  96.427s
```

## License

trie-mux is licensed under the [MIT](https://github.com/teambition/trie-mux/blob/master/LICENSE) license.
Copyright &copy; 2016-2017 [Teambition](https://www.teambition.com).