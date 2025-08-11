package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/tekig/photo-backup-server/internal/entity"
	"github.com/tekig/photo-backup-server/internal/photo"
)

var (
	errEmptyValue = errors.New("empty value")
)

type Gateway struct {
	photo   *photo.Photo
	echo    *echo.Echo
	address string
}

type GatewayConfig struct {
	Photo   *photo.Photo
	Address string
}

func New(c GatewayConfig) *Gateway {
	e := echo.New()

	g := &Gateway{
		photo:   c.Photo,
		echo:    e,
		address: c.Address,
	}

	e.Use(
		middleware.Recover(),
		middleware.Logger(),
	)

	e.GET("/content", g.hdlrContents)
	e.GET("/content/:id/original", g.hdlrContentOriginal)
	e.GET("/content/:id/thumbnail", g.hdlrContentThumbnail)
	e.POST("/content/:id", g.hdlrContentUpload)
	e.DELETE("/content/:id", g.hdlrContenDelete)

	return g
}

func (g *Gateway) Run() error {
	return g.echo.Start(g.address)
}

func (g *Gateway) Shutdown() error {
	return g.echo.Shutdown(context.TODO())
}

func (g *Gateway) hdlrContents(c echo.Context) error {
	contents, err := g.photo.Contents(c.Request().Context())
	if err != nil {
		return fmt.Errorf("contents: %w", err)
	}

	return c.JSON(http.StatusOK, contents)
}

func (g *Gateway) hdlrContentOriginal(c echo.Context) error {
	ctx := c.Request().Context()

	modifiedSince, err := fromModifiedSince(c.Request().Header.Get("If-Modified-Since"))
	if err != nil && !errors.Is(err, errEmptyValue) {
		return fmt.Errorf("modified since: %w", err)
	}

	id, err := paramID(c)
	if err != nil {
		return fmt.Errorf("param id: %w", err)
	}

	var contentRange *string
	if v := c.Request().Header.Get("Range"); v != "" {
		contentRange = &v
	}

	object, err := g.photo.ContentOriginal(ctx, entity.ObjectRequest{
		ID:              id,
		IfModifiedSince: modifiedSince,
		Range:           contentRange,
	})
	if err != nil {
		return toHTTPError(c, err)
	}
	defer object.Content.Close()

	c.Response().Header().Set("Accept-Ranges", "bytes")
	c.Response().Header().Set("Last-Modified", toModifiedSince(object.LastModified))
	var statusHTTP = http.StatusOK
	if object.ContentRange != nil {
		c.Response().Header().Set("Content-Range", *object.ContentRange)
		statusHTTP = http.StatusPartialContent
	}
	if object.ContentLength != nil {
		c.Response().Header().Set("Content-Length", strconv.Itoa(int(*object.ContentLength)))
	}

	return c.Stream(statusHTTP, object.ContentType, object.Content)
}

func (g *Gateway) hdlrContentThumbnail(c echo.Context) error {
	modifiedSince, err := fromModifiedSince(c.Request().Header.Get("If-Modified-Since"))
	if err != nil && !errors.Is(err, errEmptyValue) {
		return fmt.Errorf("modified since: %w", err)
	}

	id, err := paramID(c)
	if err != nil {
		return fmt.Errorf("param id: %w", err)
	}

	object, err := g.photo.ContentThumbnail(c.Request().Context(), id, modifiedSince)
	if err != nil {
		return toHTTPError(c, err)
	}
	defer object.Content.Close()

	c.Response().Header().Set("Last-Modified", toModifiedSince(object.LastModified))

	return c.Stream(http.StatusOK, object.ContentType, object.Content)
}

func (g *Gateway) hdlrContentUpload(c echo.Context) error {
	modifiedSince, err := fromModifiedSince(c.Request().Header.Get("Last-Modified"))
	if err != nil {
		return fmt.Errorf("modified since: %w", err)
	}

	defer c.Request().Body.Close()

	id, err := paramID(c)
	if err != nil {
		return fmt.Errorf("param id: %w", err)
	}

	if err := g.photo.ContentUpload(c.Request().Context(), entity.ObjectReader{
		Object: entity.Object{
			ID:           id,
			ContentType:  c.Request().Header.Get("Content-Type"),
			LastModified: *modifiedSince,
		},
		Content: c.Request().Body,
	}); err != nil {
		return fmt.Errorf("content upload: %w", err)
	}

	return nil
}

func (g *Gateway) hdlrContenDelete(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return fmt.Errorf("param id: %w", err)
	}

	if err := g.photo.ContentDelete(c.Request().Context(), id); err != nil {
		return fmt.Errorf("content delete: %w", err)
	}

	return nil
}

func paramID(c echo.Context) (string, error) {
	v, err := url.QueryUnescape(c.Param("id"))
	if err != nil {
		return "", fmt.Errorf("query unescape: %w", err)
	}

	return v, nil
}

func fromModifiedSince(v string) (*int64, error) {
	if v == "" {
		return nil, errEmptyValue
	}
	t, err := time.Parse(time.RFC1123, v)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	unix := t.Unix()

	return &unix, nil
}

func toModifiedSince(v int64) string {
	return time.Unix(v, 0).Format(time.RFC1123)
}

func toHTTPError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, entity.ErrNotFound):
		return echo.ErrNotFound
	case errors.Is(err, entity.ErrNotModified):
		return c.NoContent(http.StatusNotModified)
	}
	return nil
}
