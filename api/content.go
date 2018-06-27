package api

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gosimple/slug"
	yaml "gopkg.in/yaml.v2"
)

type content struct {
	Title       string            `json:"title" form:"title" query:"title" yaml:"title"`
	Slug        string            `json:"slug,omitempty" form:"slug,omitempty" query:"slug,omitempty" yaml:"slug,omitempty"`
	Body        string            `json:"body,omitempty" form:"body,omitempty" query:"body,omitempty" yaml:"body,omitempty"`
	Date        *time.Time        `json:"date,omitempty" form:"date,omitempty" query:"date,omitempty" yaml:"-"`
	PublishDate *time.Time        `json:"publishDate,omitempty" form:"publishDate,omitempty" query:"publishDate,omitempty" yaml:"-"`
	ExpiryDate  *time.Time        `json:"expiryDate,omitempty" form:"expiryDate,omitempty" query:"expiryDate,omitempty" yaml:"-"`
	Params      map[string]string `json:"params,omitempty" form:"params,omitempty" query:"params,omitempty" yaml:",inline"`
	Resources   []resource        `json:"resources,omitempty" form:"resources,omitempty" query:"resources,omitempty" yaml:"resources,omitempty"`
	Menu        menu              `json:"menu,omitempty" yaml:"menu,omitempty"`
}

type menu map[string]map[string]string

func (ct *content) encode() ([]byte, error) {
	if ct.Title == "" {
		return nil, errors.New("content must have a title")
	}
	body := ct.Body
	ct.Body = ""
	if ct.Slug == "" {
		ct.Slug = slug.Make(ct.Title)
	}
	frontMatter, err := yaml.Marshal(ct)
	ct.Body = body
	if err != nil {
		return nil, err
	}
	separator := []byte("---\n")
	return bytes.Join([][]byte{separator, frontMatter, separator, []byte(body)}, nil), nil
}

func (ct *content) decode(b []byte) error {
	parts := strings.Split(string(b), "---")
	if err := yaml.Unmarshal([]byte(parts[1]), &ct); err != nil {
		return err
	}
	ct.Body = strings.TrimSpace(strings.Join(parts[2:], ""))
	return nil
}

func (ct *content) setDates() error {
	if ct.Date != nil && !ct.Date.IsZero() {
		ct.Params["date"] = ct.Date.Local().Format(time.RFC3339Nano)
	}
	if ct.PublishDate != nil && !ct.PublishDate.IsZero() {
		ct.Params["publishdate"] = ct.PublishDate.Local().Format(time.RFC3339Nano)
	}
	if ct.ExpiryDate != nil && !ct.ExpiryDate.IsZero() {
		ct.Params["expirydate"] = ct.ExpiryDate.Local().Format(time.RFC3339Nano)
	}
	if ct.PublishDate != nil && ct.ExpiryDate != nil && !ct.ExpiryDate.IsZero() && !ct.PublishDate.IsZero() && ct.PublishDate.Unix() > ct.ExpiryDate.Unix() {
		return errors.New("publishdate do not be greater than expirydate")
	}
	return nil
}

func (ct *content) save(section string) ([]byte, error) {
	if err := ct.setDates(); err != nil {
		return nil, err
	}
	data, err := ct.encode()
	if err != nil {
		return nil, err
	}
	dir := path.Join(configuration.ContentFolder, section, ct.Slug)
	if err = os.MkdirAll(dir, 0750); err != nil && !os.IsExist(err) {
		return nil, errors.New("error creating folder to content")
	}
	f, err := os.Create(path.Join(dir, "index.md"))
	if err != nil {
		return nil, errors.New("error creating content file")
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return nil, errors.New("error writing content to file")
	}
	return data, nil
}

func (ct *content) load(section, slug string) error {
	file := path.Join(configuration.ContentFolder, section, slug, "index.md")
	f, err := ioutil.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("content not found")
		}
		return errors.New("error reading file content")
	}
	if err := ct.decode(f); err != nil {
		return err
	}
	return nil
}

func (ct *content) remove(section, slug string) error {
	if section == "" {
		return errors.New("path incomplete")
	}
	dir := path.Join(configuration.ContentFolder, section, slug)
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	return nil
}
