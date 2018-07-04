package api

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/gosimple/slug"
	"github.com/hitalos/send2hugo/config"
	"github.com/labstack/echo"
)

var configuration config.Config

// Routes to handle send2hugo request
func Routes(g *echo.Group, config config.Config) {
	configuration = config
	g.GET("", getInfo)
	g.GET("/sections", listSections)
	g.GET("/content/:section", listContents)
	g.POST("/content/:section", newContent)
	g.GET("/content/:section/:slug", getContent)
	g.PUT("/content/:section/:slug", updateContent)
	g.DELETE("/content/:section/:slug", removeContent)
	g.POST("/content/:section/:slug/attach", newResource)
	g.GET("/content/:section/:slug/:attach", getResource)
	g.DELETE("/content/:section/:slug/:attach", removeResource)
}

func getInfo(c echo.Context) error {
	msg := make(map[string]string)
	msg["info"] = "Content publishing API for Hugo"
	return c.JSON(http.StatusOK, msg)
}

func listSections(c echo.Context) error {
	entries, err := ioutil.ReadDir(configuration.ContentFolder)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	sections := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			sections = append(sections, entry.Name())
		}
	}
	return c.JSON(http.StatusOK, sections)
}

func listContents(c echo.Context) error {
	entries, err := ioutil.ReadDir(path.Join(configuration.ContentFolder, c.Param("section")))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	sections := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			sections = append(sections, entry.Name())
		}
	}
	return c.JSON(http.StatusOK, sections)
}

func newContent(c echo.Context) error {
	ct := new(content)
	if err := c.Bind(ct); err != nil {
		if err == echo.ErrUnsupportedMediaType {
			return echo.ErrUnsupportedMediaType
		}
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("error binding request data: %v", err))
	}
	section := c.Param("section")
	dir := path.Join(configuration.ContentFolder, section, slug.Make(ct.Title))
	if _, err := os.Stat(dir); err == nil {
		return echo.NewHTTPError(http.StatusConflict, "content already exists")
	}
	data, err := ct.save(section)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if strings.Contains(c.Request().Header.Get("Accept"), "markdown") {
		return c.Blob(http.StatusCreated, "text/markdown", data)
	}
	return c.JSON(http.StatusCreated, ct)
}

func updateContent(c echo.Context) error {
	ct := new(content)
	ct.load(c.Param("section"), c.Param("slug"))
	originalSlug := ct.Slug
	if err := c.Bind(ct); err != nil {
		if err == echo.ErrUnsupportedMediaType {
			return echo.ErrUnsupportedMediaType
		}
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("error binding request data: %v", err))
	}
	ct.Slug = originalSlug
	data, err := ct.save(c.Param("section"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if strings.Contains(c.Request().Header.Get("Accept"), "markdown") {
		return c.Blob(http.StatusOK, "text/markdown", data)
	}
	return c.JSON(http.StatusOK, ct)
}

func getContent(c echo.Context) error {
	ct := new(content)
	if err := ct.load(c.Param("section"), c.Param("slug")); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}
	if strings.Contains(c.Request().Header.Get("Accept"), "markdown") {
		enc, _ := ct.encode()
		return c.Blob(http.StatusCreated, "text/markdown", enc)
	}
	return c.JSON(http.StatusOK, ct)
}

func removeContent(c echo.Context) error {
	section := c.Param("section")
	slug := c.Param("slug")
	ct := new(content)

	if err := ct.load(section, slug); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if err := ct.remove(section, slug); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.String(http.StatusNoContent, "")
}

func newResource(c echo.Context) error {
	ct := new(content)
	if err := ct.load(c.Param("section"), c.Param("slug")); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	attach, err := c.FormFile("attach")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "error reading upload data on request")
	}

	f, err := attach.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "error reading upload file")
	}
	defer f.Close()

	buffer := make([]byte, 512)
	if _, err := f.Read(buffer); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "error detecting upload mimetype")
	}

	mimeTypeReject := true
	for _, mt := range configuration.MimeTypes {
		if mt == http.DetectContentType(buffer) {
			mimeTypeReject = false
		}
	}
	if mimeTypeReject {
		return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("mimetype forbidden '%s'", http.DetectContentType(buffer)))
	}
	f.Seek(0, 0)

	dir := path.Join(configuration.ContentFolder, c.Param("section"), c.Param("slug"))
	dst, err := os.Create(path.Join(dir, attach.Filename))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "error creating file on disk")
	}
	defer dst.Close()

	if _, err := io.Copy(dst, f); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "error copying attach content to disk")
	}
	r := resource{}
	r.Src = attach.Filename
	r.Title = c.FormValue("title")
	attachUpdate := false
	for i, item := range ct.Resources {
		if item.Src == attach.Filename {
			attachUpdate = true
			ct.Resources[i] = r
			break
		}
	}
	if !attachUpdate {
		ct.Resources = append(ct.Resources, r)
	}
	data, err := ct.save(c.Param("section"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if strings.Contains(c.Request().Header.Get("Accept"), "markdown") {
		return c.Blob(http.StatusCreated, "text/markdown", data)
	}

	return c.JSON(http.StatusOK, ct)
}

func getResource(c echo.Context) error {
	file := path.Join(configuration.ContentFolder, c.Param("section"), c.Param("slug"), c.Param("attach"))
	return c.File(file)
}

func removeResource(c echo.Context) error {
	dir := path.Join(configuration.ContentFolder, c.Param("section"), c.Param("slug"))
	if err := os.Remove(path.Join(dir, c.Param("attach"))); err != nil {
		if os.IsNotExist(err) {
			return echo.NewHTTPError(http.StatusNotFound, "resource not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	ct := new(content)

	if err := ct.load(c.Param("section"), c.Param("slug")); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	for i, item := range ct.Resources {
		if item.Src == c.Param("attach") {
			ct.Resources = append(ct.Resources[:i], ct.Resources[i+1:]...)
			break
		}
	}
	if _, err := ct.save(c.Param("section")); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.String(http.StatusNoContent, "")
}
