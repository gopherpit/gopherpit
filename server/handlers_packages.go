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

func (s Server) packageResolverHandler(w http.ResponseWriter, r *http.Request) {
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
		s.packageAccessLogger.Logf(level, "%s \"%s\" %s %s %s %d %f \"%s\" \"%s\"", r.RemoteAddr, xips, r.Method, httputils.GetRequestEndpoint(r)+r.URL.String(), r.Proto, code, time.Since(startTime).Seconds(), referrer, userAgent)
	}(time.Now())

	domain, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		domain = r.Host
	}
	path := domain + r.URL.Path
	resolution, err := s.PackagesService.ResolvePackage(path)
	if err != nil {
		if err == packages.DomainNotFound || err == packages.PackageNotFound {
			code = http.StatusNotFound
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(code)
			fmt.Fprintln(w, fmt.Sprintf("%s: package %s", http.StatusText(code), path))
			return
		}
		s.logger.Errorf("package resolver: resolve package: %s", err)
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
		if s.RedirectToHTTPS {
			repoRoot = "https://" + resolution.ImportPrefix
		}
	}

	s.respond(w, tidPackageResolution, map[string]interface{}{
		"GoImport":    fmt.Sprintf("%s %s %s", resolution.ImportPrefix, resolution.VCS, repoRoot),
		"GoSource":    resolution.GoSource,
		"RedirectURL": resolution.RedirectURL,
	})
}

func (s Server) packageGitUploadPackHandler(w http.ResponseWriter, r *http.Request) (notFound bool) {
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
			s.packageAccessLogger.Logf(level, "%s \"%s\" %s %s %s %d %f \"%s\" \"%s\"", r.RemoteAddr, xips, r.Method, httputils.GetRequestEndpoint(r)+r.URL.String(), r.Proto, code, time.Since(startTime).Seconds(), referrer, userAgent)
		}
	}(time.Now())

	domain, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		domain = r.Host
	}
	path := domain + strings.TrimSuffix(r.URL.Path, "/git-upload-pack")
	resolution, err := s.PackagesService.ResolvePackage(path)
	if err != nil {
		if err == packages.DomainNotFound || err == packages.PackageNotFound {
			notFound = true
			return
		}
		s.logger.Errorf("package git upload pack: resolve package: %s", err)
		code = 500
		textServerError(w, err)
		return
	}

	if resolution.Disabled {
		notFound = true
		return
	}

	code = 200
	w.Header().Set("Location", resolution.RepoRoot+"/git-upload-pack")
	w.WriteHeader(http.StatusMovedPermanently)
	return
}

var (
	errRefNotFound = errors.New("referenece not found")
	httpClient     = &http.Client{
		Timeout: 15 * time.Second,
	}
)

func (s Server) packageGitInfoRefsHandler(w http.ResponseWriter, r *http.Request) (notFound bool) {
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
			s.packageAccessLogger.Logf(level, "%s \"%s\" %s %s %s %d %f \"%s\" \"%s\"", r.RemoteAddr, xips, r.Method, httputils.GetRequestEndpoint(r)+r.URL.String(), r.Proto, code, time.Since(startTime).Seconds(), referrer, userAgent)
		}
	}(time.Now())

	domain, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		domain = r.Host
	}
	path := domain + strings.TrimSuffix(r.URL.Path, "/info/refs")
	resolution, err := s.PackagesService.ResolvePackage(path)
	if err != nil {
		if err == packages.DomainNotFound || err == packages.PackageNotFound {
			notFound = true
			return
		}
		s.logger.Errorf("package git info refs: resolve package: %s", err)
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
		s.logger.Errorf("package git info refs: http get: %s", err)
		code = 500
		textServerError(w, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		s.logger.Warningf("package git info refs: http get: %s: status code %v", refsURL, resp.StatusCode)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(resp.StatusCode)
		fmt.Fprintln(w, resp.Status)
		code = resp.StatusCode
		return
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		s.logger.Errorf("package git info refs: http read body: %s", err)
		code = 500
		textServerError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")

	if err = writeAlteredGitInfoRef(data, w, resolution.RefType, resolution.RefName); err != nil {
		if err == errRefNotFound {
			s.logger.Warningf("package git info refs: alter refs: %s", err)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, fmt.Sprintf("%s: %s %s", http.StatusText(http.StatusNotFound), resolution.RefType, resolution.RefName))
			code = 404
			return
		}
		s.logger.Errorf("package git info refs: alter refs: %s", err)
		code = 500
		textServerError(w, err)
		return
	}
	code = 200
	return
}

func writeAlteredGitInfoRef(data []byte, w io.Writer, refType, refName string) error {
	var ref string
	switch refType {
	case "branch":
		ref = "refs/heads/" + refName
	case "tag":
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
			case "branch":
				break
			case "tag":
				continue
			}
		}
		// Anotated tags
		if refType == "tag" && name == ref+"^{}" {
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
	if refType == "branch" {
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
