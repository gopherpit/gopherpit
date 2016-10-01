package httphandlers

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"text/template"
)

// DebugIndexHandler serves pprof profiles under a defined Path.
// Path must end with the slash "/".
type DebugIndexHandler struct {
	Path string
}

// ServeHTTP serves http request.
func (h DebugIndexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, h.Path) {
		name := strings.TrimPrefix(r.URL.Path, h.Path)
		if name != "" {
			debugHandler(name).ServeHTTP(w, r)
			return
		}
	}

	profiles := pprof.Profiles()
	if err := debugIndexTmpl.Execute(w, struct {
		Profiles []*pprof.Profile
		Path     string
	}{
		Profiles: profiles,
		Path:     h.Path,
	}); err != nil {
		log.Printf("debug intex handler: %s", err)
	}
}

var debugIndexTmpl = template.Must(template.New("index").Parse(`<html>
	<head>
		<title>{{.Path}}</title>
	</head>
	<body>
		profiles:<br>
		<table>
		{{range .Profiles}}
		<tr><td align=right>{{.Count}}<td><a href="{{$.Path}}{{.Name}}?debug=1">{{.Name}}</a>
		{{end}}
		</table>
		<br>
		<a href="{{.Path}}goroutine?debug=2">full goroutine stack dump</a><br>
	</body>
</html>`))

type debugHandler string

func (name debugHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	debug, _ := strconv.Atoi(r.FormValue("debug"))
	p := pprof.Lookup(string(name))
	if p == nil {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Unknown profile: %s\n", name)
		return
	}
	gc, _ := strconv.Atoi(r.FormValue("gc"))
	if name == "heap" && gc > 0 {
		runtime.GC()
	}
	p.WriteTo(w, debug)
	return
}
