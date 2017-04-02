package server

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"resenje.org/httputils"
	"resenje.org/logging"

	"gopherpit.com/gopherpit/services/packages"
)

func packageResolverHandler(w http.ResponseWriter, r *http.Request) {
	var code int
	defer func(startTime time.Time) {
		referrer := r.Referer()
		if referrer == "" {
			referrer = "-"
		}
		userAgent := r.UserAgent()
		if userAgent == "" {
			userAgent = "-"
		}
		ips := []string{}
		xfr := r.Header.Get("X-Forwarded-For")
		if xfr != "" {
			ips = append(ips, xfr)
		}
		xri := r.Header.Get("X-Real-Ip")
		if xri != "" {
			ips = append(ips, xri)
		}
		xips := "-"
		if len(ips) > 0 {
			xips = strings.Join(ips, ", ")
		}
		var level logging.Level
		switch {
		case code >= 500:
			level = logging.ERROR
		case code >= 400:
			level = logging.WARNING
		case code >= 300:
			level = logging.INFO
		case code >= 200:
			level = logging.INFO
		default:
			level = logging.DEBUG
		}
		srv.PackageAccessLogger.Logf(level, "%s \"%s\" %s %s %s %d %f \"%s\" \"%s\"", r.RemoteAddr, xips, r.Method, httputils.GetRequestEndpoint(r)+r.URL.String(), r.Proto, code, time.Since(startTime).Seconds(), referrer, userAgent)
	}(time.Now())

	domain, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		domain = r.Host
	}
	path := domain + r.URL.Path
	resolution, err := srv.PackagesService.ResolvePackage(path)
	if err != nil {
		if err == packages.ErrDomainNotFound || err == packages.ErrPackageNotFound {
			code = http.StatusNotFound
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(code)
			fmt.Fprintln(w, fmt.Sprintf("%s: package %s", http.StatusText(code), path))
			return
		}
		srv.Logger.Errorf("package resolver: resolve package: %s", err)
		code = http.StatusInternalServerError
		textServerError(w, err)
		return
	}

	if resolution.Disabled {
		code = http.StatusNotFound
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(code)
		fmt.Fprintln(w, fmt.Sprintf("%s: package %s", http.StatusText(code), path))
		return
	}

	code = 200

	repoRoot := resolution.RepoRoot
	if resolution.RefType != "" {
		// If Reference is changed, repo root does not have scheme,
		// so it is set here.
		repoRoot = "http://" + resolution.ImportPrefix
		if srv.tlsEnabled {
			repoRoot = "https://" + resolution.ImportPrefix
		}
	}

	respond(w, "PackageResolution", map[string]interface{}{
		"GoImport":    fmt.Sprintf("%s %s %s", resolution.ImportPrefix, resolution.VCS, repoRoot),
		"GoSource":    resolution.GoSource,
		"RedirectURL": resolution.RedirectURL,
	})
}

func packageGitUploadPackHandler(w http.ResponseWriter, r *http.Request) (notFound bool) {
	var code int
	defer func(startTime time.Time) {
		if !notFound {
			referrer := r.Referer()
			if referrer == "" {
				referrer = "-"
			}
			userAgent := r.UserAgent()
			if userAgent == "" {
				userAgent = "-"
			}
			ips := []string{}
			xfr := r.Header.Get("X-Forwarded-For")
			if xfr != "" {
				ips = append(ips, xfr)
			}
			xri := r.Header.Get("X-Real-Ip")
			if xri != "" {
				ips = append(ips, xri)
			}
			xips := "-"
			if len(ips) > 0 {
				xips = strings.Join(ips, ", ")
			}
			var level logging.Level
			switch {
			case code >= 500:
				level = logging.ERROR
			case code >= 400:
				level = logging.WARNING
			case code >= 300:
				level = logging.INFO
			case code >= 200:
				level = logging.INFO
			default:
				level = logging.DEBUG
			}
			srv.PackageAccessLogger.Logf(level, "%s \"%s\" %s %s %s %d %f \"%s\" \"%s\"", r.RemoteAddr, xips, r.Method, httputils.GetRequestEndpoint(r)+r.URL.String(), r.Proto, code, time.Since(startTime).Seconds(), referrer, userAgent)
		}
	}(time.Now())

	domain, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		domain = r.Host
	}
	path := domain + strings.TrimSuffix(r.URL.Path, "/git-upload-pack")
	resolution, err := srv.PackagesService.ResolvePackage(path)
	if err != nil {
		if err == packages.ErrDomainNotFound || err == packages.ErrPackageNotFound {
			notFound = true
			return
		}
		srv.Logger.Errorf("package git upload pack: resolve package: %s", err)
		code = 500
		textServerError(w, err)
		return
	}

	if resolution.Disabled {
		notFound = true
		return
	}

	req, err := http.NewRequest(r.Method, strings.TrimSuffix(resolution.RepoRoot, ".git")+".git/git-upload-pack", r.Body)
	if err != nil {
		srv.Logger.Errorf("package git upload pack: new request: %s", err)
		code = 500
		textServerError(w, err)
		return
	}
	defer r.Body.Close()

	req.Header.Set("User-Agent", r.Header.Get("User-Agent"))
	req.Header.Set("Accept", r.Header.Get("Accept"))
	req.Header.Set("Content-Type", r.Header.Get("Content-Type"))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		srv.Logger.Errorf("package git upload pack: make request: %s", err)
		code = 500
		textServerError(w, err)
		return
	}
	defer resp.Body.Close()

	if _, err = io.Copy(w, resp.Body); err != nil {
		srv.Logger.Errorf("package git upload pack: copy request data: %s", err)
		code = 500
		textServerError(w, err)
		return
	}

	code = 200
	return
}

var (
	errRefNotFound = errors.New("reference not found")
	httpClient     = &http.Client{
		Timeout: 15 * time.Second,
	}
)

func packageGitInfoRefsHandler(w http.ResponseWriter, r *http.Request) (notFound bool) {
	var code int
	defer func(startTime time.Time) {
		if !notFound {
			referrer := r.Referer()
			if referrer == "" {
				referrer = "-"
			}
			userAgent := r.UserAgent()
			if userAgent == "" {
				userAgent = "-"
			}
			ips := []string{}
			xfr := r.Header.Get("X-Forwarded-For")
			if xfr != "" {
				ips = append(ips, xfr)
			}
			xri := r.Header.Get("X-Real-Ip")
			if xri != "" {
				ips = append(ips, xri)
			}
			xips := "-"
			if len(ips) > 0 {
				xips = strings.Join(ips, ", ")
			}
			var level logging.Level
			switch {
			case code >= 500:
				level = logging.ERROR
			case code >= 400:
				level = logging.WARNING
			case code >= 300:
				level = logging.INFO
			case code >= 200:
				level = logging.INFO
			default:
				level = logging.DEBUG
			}
			srv.PackageAccessLogger.Logf(level, "%s \"%s\" %s %s %s %d %f \"%s\" \"%s\"", r.RemoteAddr, xips, r.Method, httputils.GetRequestEndpoint(r)+r.URL.String(), r.Proto, code, time.Since(startTime).Seconds(), referrer, userAgent)
		}
	}(time.Now())

	domain, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		domain = r.Host
	}
	path := domain + strings.TrimSuffix(r.URL.Path, "/info/refs")
	resolution, err := srv.PackagesService.ResolvePackage(path)
	if err != nil {
		if err == packages.ErrDomainNotFound || err == packages.ErrPackageNotFound {
			notFound = true
			return
		}
		srv.Logger.Errorf("package git info refs: resolve package: %s", err)
		code = 500
		textServerError(w, err)
		return
	}
	refsURL := resolution.RepoRoot
	refsURL = strings.TrimRight(refsURL, "/")
	refsURL = strings.TrimSuffix(refsURL, ".git")
	refsURL += ".git/info/refs?service=git-upload-pack"

	resp, err := httpClient.Get(refsURL)
	if err != nil {
		srv.Logger.Errorf("package git info refs: http get: %s", err)
		code = 500
		textServerError(w, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		srv.Logger.Warningf("package git info refs: http get: %s: status code %v", refsURL, resp.StatusCode)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(resp.StatusCode)
		fmt.Fprintln(w, resp.Status)
		code = resp.StatusCode
		return
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		srv.Logger.Errorf("package git info refs: http read body: %s", err)
		code = 500
		textServerError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")

	if err = writeAlteredGitInfoRef(data, w, resolution.RefType, resolution.RefName); err != nil {
		if err == errRefNotFound {
			srv.Logger.Warningf("package git info refs: alter refs: %s", err)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, fmt.Sprintf("%s: %s %s", http.StatusText(http.StatusNotFound), resolution.RefType, resolution.RefName))
			code = 404
			return
		}
		srv.Logger.Errorf("package git info refs: alter refs: %s", err)
		code = 500
		textServerError(w, err)
		return
	}
	code = 200
	return
}

func writeAlteredGitInfoRef(data []byte, w io.Writer, refType packages.RefType, refName string) error {
	var ref string
	switch refType {
	case packages.RefTypeBranch:
		ref = "refs/heads/" + refName
	case packages.RefTypeTag:
		ref = "refs/tags/" + refName
	default:
		return fmt.Errorf("invalid reference type: %s", refType)
	}
	var (
		headIndexStart   int
		headIndexEnd     int
		masterIndexStart int
		masterIndexEnd   int
	)
	var refHash []byte
	for stat, end := 0, 0; stat < len(data); stat = end {
		size, err := strconv.ParseInt(string(data[stat:stat+4]), 16, 32)
		if err != nil {
			return fmt.Errorf("parse line size: %s", string(data[stat:stat+4]))
		}
		if size == 0 {
			size = 4
		}
		end = stat + int(size)
		if end > len(data) {
			return fmt.Errorf("incomplete data")
		}
		if len(data) > stat+4 && data[stat+4] == '#' {
			continue
		}

		hashIndexStart := stat + 4
		hashIndexEnd := bytes.IndexByte(data[hashIndexStart:end], ' ')
		if hashIndexEnd < 0 || hashIndexEnd != 40 {
			continue
		}
		hashIndexEnd += hashIndexStart

		nameIndexStart := hashIndexEnd + 1
		nameIndexEnd := bytes.IndexAny(data[nameIndexStart:end], "\n\x00")
		if nameIndexEnd < 0 {
			nameIndexEnd = end
		} else {
			nameIndexEnd += nameIndexStart
		}

		name := string(data[nameIndexStart:nameIndexEnd])
		if name == "HEAD" {
			headIndexStart = stat
			headIndexEnd = end
			continue
		}
		if name == "refs/heads/master" {
			masterIndexStart = stat
			masterIndexEnd = end
			continue
		}
		if name == ref {
			refHash = data[hashIndexStart:hashIndexEnd]
			switch refType {
			case packages.RefTypeBranch:
				break
			case packages.RefTypeTag:
				continue
			}
		}
		// Anotated tags
		if refType == packages.RefTypeTag && name == ref+"^{}" {
			refHash = data[hashIndexStart:hashIndexEnd]
			break
		}
	}

	if headIndexStart == 0 || len(refHash) == 0 {
		return errRefNotFound
	}

	w.Write(data[:headIndexStart])

	capabilities := ""
	if i := bytes.Index(data[headIndexStart:headIndexEnd], []byte{'\x00'}); i > 0 {
		capabilities = strings.Replace(string(data[headIndexStart+i+1:headIndexEnd-1]), "symref=", "oldref=", -1)
	}

	var line string
	if refType == packages.RefTypeBranch {
		// Always reference master branch in symref to be able to change branch in the future, and
		// to have go get -u working.
		if capabilities == "" {
			line = fmt.Sprintf("%s HEAD\x00symref=HEAD:refs/heads/master\n", refHash)
		} else {
			line = fmt.Sprintf("%s HEAD\x00symref=HEAD:refs/heads/master %s\n", refHash, capabilities)
		}
	} else {
		if capabilities == "" {
			line = fmt.Sprintf("%s HEAD\n", refHash)
		} else {
			line = fmt.Sprintf("%s HEAD\x00%s\n", refHash, capabilities)
		}
	}
	fmt.Fprintf(w, "%04x%s", 4+len(line), line)

	line = fmt.Sprintf("%s refs/heads/master\n", refHash)
	fmt.Fprintf(w, "%04x%s", 4+len(line), line)

	if masterIndexStart > 0 {
		w.Write(data[headIndexEnd:masterIndexStart])
		w.Write(data[masterIndexEnd:])
	} else {
		w.Write(data[headIndexEnd:])
	}

	return nil
}
