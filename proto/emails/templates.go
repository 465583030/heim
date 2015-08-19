package emails

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/textproto"
	"net/url"
	"path/filepath"
	"strings"
)

type Template string

type emailCommon interface {
	setToAddress(to string)
	AccountEmailAddress() string
	getAttachments() map[string]string
	setStaticFiles(map[string][]byte)
}

type TemplateDataCommon struct {
	LocalDomain string

	// for internal use
	toAddress   string
	attachments map[string]string
	staticFiles map[string][]byte
}

func (tdc *TemplateDataCommon) AccountEmailAddress() string { return tdc.toAddress }

func (tdc *TemplateDataCommon) File(path string) (template.URL, error) {
	// Verify file is actually available.
	if _, ok := tdc.staticFiles[path]; !ok {
		return "", fmt.Errorf("%s: file not available", path)
	}

	// Derive Content-ID from path and t.localDomain.
	cid := fmt.Sprintf("%s@%s", url.QueryEscape(path), tdc.LocalDomain)
	if tdc.attachments == nil {
		tdc.attachments = map[string]string{}
	}
	tdc.attachments[path] = cid
	return template.URL("cid:" + cid), nil
}

func (tdc *TemplateDataCommon) getAttachments() map[string]string { return tdc.attachments }

func (tdc *TemplateDataCommon) setStaticFiles(staticFiles map[string][]byte) {
	tdc.staticFiles = staticFiles
	tdc.attachments = map[string]string{}
}

func (tdc *TemplateDataCommon) setToAddress(to string) {
	tdc.toAddress = to
}

type TemplateResult struct {
	Header      textproto.MIMEHeader
	Text        []byte
	HTML        []byte
	Attachments map[string]string
}

type TemplateTest struct {
	Data      interface{}
	Validator func(result *TemplateResult) error
}

func LoadTemplates(path string) (*Templater, []error) {
	// Scan the top-level directory of the given path.
	entries, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, []error{err}
	}

	// Find and parse templates.
	templates := map[Template]*template.Template{}
	errors := []error{}
	for _, entry := range entries {
		filename := entry.Name()
		if strings.HasSuffix(filename, ".hdr") {
			tmplName := Template(filename[:len(filename)-4])
			templates[tmplName], err = template.ParseGlob(fmt.Sprintf("%s/%s.*", path, tmplName))
			if err != nil {
				errors = append(errors, err)
				continue
			}
		}
	}

	if len(errors) > 0 {
		return nil, errors
	}

	// Scan the static directory of the given path. Load all files into memory.
	entries, err = ioutil.ReadDir(filepath.Join(path, "static"))
	if err != nil {
		return nil, []error{err}
	}

	staticFiles := map[string][]byte{}

	for _, entry := range entries {
		if !entry.IsDir() {
			fpath := filepath.Join(path, "static", entry.Name())
			staticFiles[entry.Name()], err = ioutil.ReadFile(fpath)
			if err != nil {
				errors = append(errors, err)
				continue
			}
		}
	}

	if len(errors) > 0 {
		return nil, errors
	}

	templater := &Templater{
		localDomain: "localhost",
		staticFiles: staticFiles,
		templates:   templates,
	}
	return templater, nil
}

type Templater struct {
	localDomain string
	staticFiles map[string][]byte
	templates   map[Template]*template.Template
}

func (t *Templater) Validate(tmplName Template, tests ...TemplateTest) []error {
	errors := []error{}
	for i, test := range tests {
		if ec, ok := test.Data.(emailCommon); ok {
			ec.setToAddress("user@somewhere.invalid")
		}

		result, err := t.Evaluate(tmplName, test.Data)
		if err != nil {
			errors = append(errors, fmt.Errorf("%s test #%d: evaluation error: %s", tmplName, i+1, err))
			continue
		}

		if test.Validator != nil {
			if err := test.Validator(result); err != nil {
				errors = append(
					errors, fmt.Errorf("%s test #%d: validation error: %s", tmplName, i+1, err))
				continue
			}
		}
	}
	if len(errors) == 0 {
		return nil
	}
	return errors
}

func (t *Templater) Evaluate(tmplName Template, data interface{}) (*TemplateResult, error) {
	result := &TemplateResult{}

	tmpl, ok := t.templates[tmplName]
	if !ok {
		return nil, fmt.Errorf("no templates found for %s", tmplName)
	}

	if ec, ok := data.(emailCommon); ok {
		ec.setStaticFiles(t.staticFiles)
	}

	headerBytes, err := evaluate(tmpl, tmplName, "hdr", data)
	if err != nil {
		return nil, fmt.Errorf("%s.hdr: %s", tmplName, err)
	}

	r := textproto.NewReader(bufio.NewReader(bytes.NewReader(headerBytes)))
	result.Header, err = r.ReadMIMEHeader()
	if err != nil {
		return nil, fmt.Errorf("%s.hdr: %s", tmplName, err)
	}

	if result.Text, err = evaluate(tmpl, tmplName, "txt", data); err != nil {
		return nil, fmt.Errorf("%s.txt: %s", tmplName, err)
	}

	if result.HTML, err = evaluate(tmpl, tmplName, "html", data); err != nil {
		return nil, fmt.Errorf("%s.html: %s", tmplName, err)
	}

	if ec, ok := data.(emailCommon); ok {
		result.Attachments = ec.getAttachments()
	}

	return result, nil
}

func evaluate(tmpl *template.Template, tmplName Template, ext string, data interface{}) ([]byte, error) {
	w := &bytes.Buffer{}
	name := fmt.Sprintf("%s.%s", tmplName, ext)
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		return nil, err
	}
	if ext == "hdr" {
		// Ensure content ends with a blank line.
		if !bytes.HasSuffix(w.Bytes(), []byte("\n\n")) {
			w.WriteString("\n\n")
		}
	}
	return w.Bytes(), nil
}

func IsEmailCommon(x interface{}) bool {
	_, ok := x.(emailCommon)
	return ok
}
