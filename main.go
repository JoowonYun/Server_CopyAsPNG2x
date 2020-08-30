package main

import (
	"html/template"
	"io"
	"net/http"
	"time"
	"strconv"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/tylerb/graceful"
)

func main() {
	e := echo.New()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
	}))

	e.Static("/", "./views/img")
	e.File("/favicon.ico", "./views/img/favicon.png")
	renderer := &TemplateRenderer{
		templates: template.Must(template.ParseGlob("./views/*.html")),
	}
	e.Renderer = renderer

	imageMap := make(map[string]chan string)

	for i := 0; i < 100000; i++ {
		imageMap[strconv.Itoa(i)] = make(chan string, 1)
	}

	e.POST("/copyaspng2x/image", func(c echo.Context) error {
		hash := c.FormValue("hash")
		image := c.FormValue("image")

		imageMap[hash] <-image

		println("POST / " + c.RealIP() + " / " + hash)
		return c.String(http.StatusOK, "")
	})

	e.GET("/copyaspng2x/view", func(c echo.Context) error {
		time.After(5 * time.Second)
		hash := c.QueryParams().Get("hash")
		width := c.QueryParams().Get("width")

		var imageCh chan string

		defer func() error {
			ch, exist := imageMap[hash]
			if exist {
				close(ch)
				delete(imageMap, hash)
			}

			imageMap[hash] = make(chan string, 1)

			if !exist {
				return c.Render(http.StatusOK, "404.html", nil)
			}

			return nil
		}()

		image := ""

		timeoutCh := time.After(5 * time.Second)
		imageCh, _ = imageMap[hash]
		println("GET / " + c.RealIP() + " / " + hash)
		select {
		case <-timeoutCh:
			println("GET - Time out / ", c.RealIP(), " / ", hash)
			return c.Render(http.StatusOK, "404.html", nil)
		case image = <-imageCh:
		}

		return c.Render(http.StatusOK, "image.html", map[string]interface{}{
			"width": width,
			"image": image,
		})
	})

	e.GET("/copyaspng2x/dialog", func(c echo.Context) error {
		return c.Render(http.StatusOK, "dialog.html", nil)
	})

	certfile := "/etc/letsencrypt/live/figma.joowonyun.space/fullchain.pem"
	keyfile := "/etc/letsencrypt/live/figma.joowonyun.space/privkey.pem"
	e.TLSServer.Addr = ":443"
	graceful.ListenAndServeTLS(e.TLSServer, certfile, keyfile, 10*time.Second)
}

// TemplateRenderer is a custom html/template renderer for Echo framework
type TemplateRenderer struct {
	templates *template.Template
}

// Render renders a template document
func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {

	// Add global methods if data is a map
	if viewContext, isMap := data.(map[string]interface{}); isMap {
		viewContext["reverse"] = c.Echo().Reverse
	}

	return t.templates.ExecuteTemplate(w, name, data)
}
