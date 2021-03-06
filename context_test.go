package akita

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"text/template"
	"time"

	"strings"

	"net/url"

	"encoding/xml"

	"github.com/stretchr/testify/assert"
)

type (
	Template struct {
		templates *template.Template
	}
)

func (t *Template) Render(w io.Writer, name string, data interface{}, ctx Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func TestContext(t *testing.T) {
	a := New()
	req := httptest.NewRequest(POST, "/", strings.NewReader(userJSON))
	rec := httptest.NewRecorder()
	ctx := a.NewContext(req, rec).(*context)

	// Echo
	assert.Equal(t, a, ctx.Akita())

	// Request
	assert.NotNil(t, ctx.Request())

	// Response
	assert.NotNil(t, ctx.Response())

	//--------
	// Render
	//--------

	tmpl := &Template{
		templates: template.Must(template.New("hello").Parse("Hello, {{.}}!")),
	}
	ctx.akita.Renderer = tmpl
	err := ctx.Render(http.StatusOK, "hello", "Jon Snow")
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "Hello, Jon Snow!", rec.Body.String())
	}

	ctx.akita.Renderer = nil
	err = ctx.Render(http.StatusOK, "hello", "Jon Snow")
	assert.Error(t, err)

	// JSON
	rec = httptest.NewRecorder()
	ctx = a.NewContext(req, rec).(*context)
	err = ctx.JSON(http.StatusOK, user{1, "Jon Snow"})
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, MIMEApplicationJSONCharsetUTF8, rec.Header().Get(HeaderContentType))
		assert.Equal(t, userJSON, rec.Body.String())
	}

	// JSON with "?pretty"
	req = httptest.NewRequest(GET, "/?pretty", nil)
	rec = httptest.NewRecorder()
	ctx = a.NewContext(req, rec).(*context)
	err = ctx.JSON(http.StatusOK, user{1, "Jon Snow"})
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, MIMEApplicationJSONCharsetUTF8, rec.Header().Get(HeaderContentType))
		assert.Equal(t, userJSONPretty, rec.Body.String())
	}
	req = httptest.NewRequest(GET, "/", nil) // reset

	// JSONPretty
	rec = httptest.NewRecorder()
	ctx = a.NewContext(req, rec).(*context)
	err = ctx.JSONPretty(http.StatusOK, user{1, "Jon Snow"}, "  ")
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, MIMEApplicationJSONCharsetUTF8, rec.Header().Get(HeaderContentType))
		assert.Equal(t, userJSONPretty, rec.Body.String())
	}

	// JSON (error)
	rec = httptest.NewRecorder()
	ctx = a.NewContext(req, rec).(*context)
	err = ctx.JSON(http.StatusOK, make(chan bool))
	assert.Error(t, err)

	// JSONP
	rec = httptest.NewRecorder()
	ctx = a.NewContext(req, rec).(*context)
	callback := "callback"
	err = ctx.JSONP(http.StatusOK, callback, user{1, "Jon Snow"})
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, MIMEApplicationJavaScriptCharsetUTF8, rec.Header().Get(HeaderContentType))
		assert.Equal(t, callback+"("+userJSON+");", rec.Body.String())
	}

	// XML
	rec = httptest.NewRecorder()
	ctx = a.NewContext(req, rec).(*context)
	err = ctx.XML(http.StatusOK, user{1, "Jon Snow"})
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, MIMEApplicationXMLCharsetUTF8, rec.Header().Get(HeaderContentType))
		assert.Equal(t, xml.Header+userXML, rec.Body.String())
	}

	// XML with "?pretty"
	req = httptest.NewRequest(GET, "/?pretty", nil)
	rec = httptest.NewRecorder()
	ctx = a.NewContext(req, rec).(*context)
	err = ctx.XML(http.StatusOK, user{1, "Jon Snow"})
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, MIMEApplicationXMLCharsetUTF8, rec.Header().Get(HeaderContentType))
		assert.Equal(t, xml.Header+userXMLPretty, rec.Body.String())
	}
	req = httptest.NewRequest(GET, "/", nil)

	// XML (error)
	rec = httptest.NewRecorder()
	ctx = a.NewContext(req, rec).(*context)
	err = ctx.XML(http.StatusOK, make(chan bool))
	assert.Error(t, err)

	// XMLPretty
	rec = httptest.NewRecorder()
	ctx = a.NewContext(req, rec).(*context)
	err = ctx.XMLPretty(http.StatusOK, user{1, "Jon Snow"}, "  ")
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, MIMEApplicationXMLCharsetUTF8, rec.Header().Get(HeaderContentType))
		assert.Equal(t, xml.Header+userXMLPretty, rec.Body.String())
	}

	// String
	rec = httptest.NewRecorder()
	ctx = a.NewContext(req, rec).(*context)
	err = ctx.String(http.StatusOK, "Hello, World!")
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, MIMETextPlainCharsetUTF8, rec.Header().Get(HeaderContentType))
		assert.Equal(t, "Hello, World!", rec.Body.String())
	}

	// HTML
	rec = httptest.NewRecorder()
	ctx = a.NewContext(req, rec).(*context)
	err = ctx.HTML(http.StatusOK, "Hello, <strong>World!</strong>")
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, MIMETextHTMLCharsetUTF8, rec.Header().Get(HeaderContentType))
		assert.Equal(t, "Hello, <strong>World!</strong>", rec.Body.String())
	}

	// Stream
	rec = httptest.NewRecorder()
	ctx = a.NewContext(req, rec).(*context)
	r := strings.NewReader("response from a stream")
	err = ctx.Stream(http.StatusOK, "application/octet-stream", r)
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/octet-stream", rec.Header().Get(HeaderContentType))
		assert.Equal(t, "response from a stream", rec.Body.String())
	}

	// Attachment
	rec = httptest.NewRecorder()
	ctx = a.NewContext(req, rec).(*context)
	err = ctx.Attachment("_fixture/images/akita.png", "akita.png")
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "attachment; filename=\"akita.png\"", rec.Header().Get(HeaderContentDisposition))
		assert.Equal(t, 45619, rec.Body.Len())
	}

	// Inline
	rec = httptest.NewRecorder()
	ctx = a.NewContext(req, rec).(*context)
	err = ctx.Inline("_fixture/images/akita.png", "akita.png")
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "inline; filename=\"akita.png\"", rec.Header().Get(HeaderContentDisposition))
		assert.Equal(t, 45619, rec.Body.Len())
	}

	// NoContent
	rec = httptest.NewRecorder()
	ctx = a.NewContext(req, rec).(*context)
	ctx.NoContent(http.StatusOK)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Error
	rec = httptest.NewRecorder()
	ctx = a.NewContext(req, rec).(*context)
	ctx.Error(errors.New("error"))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	// Reset
	ctx.SetParamNames("foo")
	ctx.SetParamValues("bar")
	ctx.Set("foe", "ban")
	ctx.query = url.Values(map[string][]string{"fon": []string{"baz"}})
	ctx.Reset(req, httptest.NewRecorder())
	assert.Equal(t, 0, len(ctx.ParamValues()))
	assert.Equal(t, 0, len(ctx.ParamNames()))
	assert.Equal(t, 0, len(ctx.store))
	assert.Equal(t, "", ctx.Path())
	assert.Equal(t, 0, len(ctx.QueryParams()))
}

func TestContextCookie(t *testing.T) {
	e := New()
	req := httptest.NewRequest(GET, "/", nil)
	theme := "theme=light"
	user := "user=Jon Snow"
	req.Header.Add(HeaderCookie, theme)
	req.Header.Add(HeaderCookie, user)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec).(*context)

	// Read single
	cookie, err := c.Cookie("theme")
	if assert.NoError(t, err) {
		assert.Equal(t, "theme", cookie.Name)
		assert.Equal(t, "light", cookie.Value)
	}

	// Read multiple
	for _, cookie := range c.Cookies() {
		switch cookie.Name {
		case "theme":
			assert.Equal(t, "light", cookie.Value)
		case "user":
			assert.Equal(t, "Jon Snow", cookie.Value)
		}
	}

	// Write
	cookie = &http.Cookie{
		Name:     "SSID",
		Value:    "Ap4PGTEq",
		Domain:   "liusha.me",
		Path:     "/",
		Expires:  time.Now(),
		Secure:   true,
		HttpOnly: true,
	}
	c.SetCookie(cookie)
	assert.Contains(t, rec.Header().Get(HeaderSetCookie), "SSID")
	assert.Contains(t, rec.Header().Get(HeaderSetCookie), "Ap4PGTEq")
	assert.Contains(t, rec.Header().Get(HeaderSetCookie), "liusha.me")
	assert.Contains(t, rec.Header().Get(HeaderSetCookie), "Secure")
	assert.Contains(t, rec.Header().Get(HeaderSetCookie), "HttpOnly")
}

func TestContextPath(t *testing.T) {
	e := New()
	r := e.Router()

	r.Add(GET, "/users/:id", nil)
	c := e.NewContext(nil, nil)
	r.Find(GET, "/users/1", c)
	assert.Equal(t, "/users/:id", c.Path())

	r.Add(GET, "/users/:uid/files/:fid", nil)
	c = e.NewContext(nil, nil)
	r.Find(GET, "/users/1/files/1", c)
	assert.Equal(t, "/users/:uid/files/:fid", c.Path())
}

func TestContextPathParam(t *testing.T) {
	e := New()
	req := httptest.NewRequest(GET, "/", nil)
	c := e.NewContext(req, nil)

	// ParamNames
	c.SetParamNames("uid", "fid")
	assert.EqualValues(t, []string{"uid", "fid"}, c.ParamNames())

	// ParamValues
	c.SetParamValues("101", "501")
	assert.EqualValues(t, []string{"101", "501"}, c.ParamValues())

	// Param
	assert.Equal(t, "501", c.Param("fid"))
}

func TestContextPathParamNamesAlais(t *testing.T) {
	e := New()
	req := httptest.NewRequest(GET, "/", nil)
	c := e.NewContext(req, nil)

	c.SetParamNames("id,name")
	c.SetParamValues("joe")

	assert.Equal(t, "joe", c.Param("id"))
	assert.Equal(t, "joe", c.Param("name"))
}

func TestContextFormValue(t *testing.T) {
	f := make(url.Values)
	f.Set("name", "Jon Snow")
	f.Set("email", "jon@liusha.me")

	e := New()
	req := httptest.NewRequest(POST, "/", strings.NewReader(f.Encode()))
	req.Header.Add(HeaderContentType, MIMEApplicationForm)
	c := e.NewContext(req, nil)

	// FormValue
	assert.Equal(t, "Jon Snow", c.FormValue("name"))
	assert.Equal(t, "jon@liusha.me", c.FormValue("email"))

	// FormParams
	params, err := c.FormParams()
	if assert.NoError(t, err) {
		assert.Equal(t, url.Values{
			"name":  []string{"Jon Snow"},
			"email": []string{"jon@liusha.me"},
		}, params)
	}
}

func TestContextQueryParam(t *testing.T) {
	q := make(url.Values)
	q.Set("name", "Jon Snow")
	q.Set("email", "jon@liusha.me")
	req := httptest.NewRequest(GET, "/?"+q.Encode(), nil)
	e := New()
	c := e.NewContext(req, nil)

	// QueryParam
	assert.Equal(t, "Jon Snow", c.QueryParam("name"))
	assert.Equal(t, "jon@liusha.me", c.QueryParam("email"))

	// QueryParams
	assert.Equal(t, url.Values{
		"name":  []string{"Jon Snow"},
		"email": []string{"jon@liusha.me"},
	}, c.QueryParams())
}

func TestContextFormFile(t *testing.T) {
	e := New()
	buf := new(bytes.Buffer)
	mr := multipart.NewWriter(buf)
	w, err := mr.CreateFormFile("file", "test")
	if assert.NoError(t, err) {
		w.Write([]byte("test"))
	}
	mr.Close()
	req := httptest.NewRequest(POST, "/", buf)
	req.Header.Set(HeaderContentType, mr.FormDataContentType())
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	f, err := c.FormFile("file")
	if assert.NoError(t, err) {
		assert.Equal(t, "test", f.Filename)
	}
}

func TestContextMultipartForm(t *testing.T) {
	e := New()
	buf := new(bytes.Buffer)
	mw := multipart.NewWriter(buf)
	mw.WriteField("name", "Jon Snow")
	mw.Close()
	req := httptest.NewRequest(POST, "/", buf)
	req.Header.Set(HeaderContentType, mw.FormDataContentType())
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	f, err := c.MultipartForm()
	if assert.NoError(t, err) {
		assert.NotNil(t, f)
	}
}

func TestContextRedirect(t *testing.T) {
	e := New()
	req := httptest.NewRequest(GET, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	assert.Equal(t, nil, c.Redirect(http.StatusMovedPermanently, "https://liusha.me/tags/akita"))
	assert.Equal(t, http.StatusMovedPermanently, rec.Code)
	assert.Equal(t, "https://liusha.me/tags/akita", rec.Header().Get(HeaderLocation))
	assert.Error(t, c.Redirect(310, "https://liusha.me/tags/akita"))
}

func TestContextStore(t *testing.T) {
	var c Context
	c = new(context)
	c.Set("name", "Jon Snow")
	assert.Equal(t, "Jon Snow", c.Get("name"))
}

func TestContextHandler(t *testing.T) {
	e := New()
	r := e.Router()
	b := new(bytes.Buffer)

	r.Add(GET, "/handler", func(Context) error {
		_, err := b.Write([]byte("handler"))
		return err
	})
	c := e.NewContext(nil, nil)
	r.Find(GET, "/handler", c)
	c.Handler()(c)
	assert.Equal(t, "handler", b.String())
}
