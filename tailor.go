package tailor

import (
	"bytes"
	"errors"
	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
	"github.com/mholt/caddy/caddyhttp/httpserver"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"time"
)

type Tailor struct {
	Next httpserver.Handler
}

type Fragment struct {
	Src         string
	FallbackSrc string
	Timeout     time.Duration
	ID          string
	IsPublic    bool
	IsPrimary   bool
	IsAsync     bool
	Method      string
}

func readFragment(e *goquery.Selection) (Fragment, error) {
	var f Fragment
	id, hasID := e.Attr("id")
	if !hasID {
		var bid []byte
		uid := uuid.New()
		uid.UnmarshalText(bid)
		id = string(bid)
	}
	f.ID = id

	src, hasSrc := e.Attr("src")
	if !hasSrc {
		return f, errors.New("src-Attribute is required")
	}
	f.Src = src
	method := e.AttrOr("method", "GET")
	f.Method = strings.ToUpper(method)

	timeoutStr, hasTimeout := e.Attr("timeout")
	if hasTimeout {
		timeout, timeoutErr := strconv.Atoi(timeoutStr)
		f.Timeout = time.Millisecond * time.Duration(timeout)
	} else {
		f.Timeout = time.Second * 60
	}

	fallbackSrc, hasFallbackSrc := e.Attr("fallback-src")
	if hasFallbackSrc {
		f.FallbackSrc = fallbackSrc
	}

	_, isPrimary := e.Attr("primary")
	f.IsPrimary = isPrimary

	_, isPublic := e.Attr("public")
	f.IsPublic = isPublic

	_, isAsync := e.Attr("async")
	f.IsAsync = isAsync

	return f, nil
}

func doRequest(method string, src string, timeout time.Duration, fallbackSrc string) (*http.Response, error) {
	// Create a new request from method and src.
	req, _ := http.NewRequest(method, src, nil)
	client := &http.Client{Timeout: timeout}
	resp, reqErr := client.Do(req)
	if reqErr != nil && fallbackSrc != "" {
		fbReq, _ := http.NewRequest(method, fallbackSrc, nil)
		return client.Do(fbReq)
	}
	return resp, reqErr
}

func (t Tailor) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {

	rec := httptest.NewRecorder()
	status, err := t.Next.ServeHTTP(rec, r)

	// Early exit
	if status != http.StatusOK {
		return status, err
	}

	// Read all from the response recorder and create a new goquery
	// document from it to search an replace fragments.
	reader := strings.NewReader(string(rec.Body.Bytes()))
	doc, docErr := goquery.NewDocumentFromReader(reader)

	if docErr != nil {
		return http.StatusInternalServerError, docErr
	}

	doc.Find("fragment").Each(func(i int, elem *goquery.Selection) {
		f, err := readFragment(elem)

		// Error reading all fragment options.
		// Exit here.
		if err != nil {
			elem.ReplaceWithHtml(err.Error())
			return
		}

		resp, respErr := doRequest(f.Method, f.Src, f.Timeout, f.FallbackSrc)

		if f.IsPrimary {
			status = resp.StatusCode
		}
		defer resp.Body.Close()
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		elem.ReplaceWithHtml(buf.String())

	})

	newContent, _ := doc.Html()
	contentLength := len(newContent)

	w.Header().Set("Content-Length", strconv.Itoa(contentLength))

	w.Write([]byte(newContent))
	return status, nil
}
