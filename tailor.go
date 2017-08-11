package tailor

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
	"github.com/mholt/caddy/caddyhttp/httpserver"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
)

type Tailor struct {
	Next httpserver.Handler
}

type Fragment struct {
	Src       string
	ID        string
	IsPublic  bool
	IsPrimary bool
	IsAsync   bool
	Method    string
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

	_, isPrimary := e.Attr("primary")
	f.IsPrimary = isPrimary

	_, isPublic := e.Attr("public")
	f.IsPublic = isPublic

	_, isAsync := e.Attr("async")
	f.IsAsync = isAsync

	return f, nil
}

func (t Tailor) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {

	rec := httptest.NewRecorder()
	status, err := t.Next.ServeHTTP(rec, r)

	// Early exit
	if status != http.StatusOK {
		return status, err
	}

	reader := strings.NewReader(string(rec.Body.Bytes()))
	doc, docErr := goquery.NewDocumentFromReader(reader)

	if docErr != nil {
		return http.StatusInternalServerError, docErr
	}

	doc.Find("fragment").Each(func(i int, elem *goquery.Selection) {
		f, err := readFragment(elem)

		if err != nil {
			elem.ReplaceWithHtml(err.Error())
			return
		}

		req, _ := http.NewRequest(f.Method, f.Src, nil)

		if f.IsAsync {
			// TODO: async
		} else {
			if f.IsPublic {
				for k, v := range r.Header {
					req.Header.Add(k, v[0])
				}
			}
		}

		resp, _ := http.DefaultClient.Do(req)
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
