package main

import (
	"html/template"
	"io"
	"net/http"
	"time"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/tylerb/graceful"
)

func main() {
	e := echo.New()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
	}))

	renderer := &TemplateRenderer{
		templates: template.Must(template.ParseGlob("./views/*.html")),
	}
	e.Renderer = renderer

	imageMap := make(map[string]chan string)

	for i := 0; i < 100000; i++ {
		imageMap[string(i)] = make(chan string, 1)
	}

	e.POST("/copyaspng2x/image", func(c echo.Context) error {
		hash := c.FormValue("hash")
		image := c.FormValue("image")

		imageMap[hash] <- image

		println("POST / " + c.RealIP() + " / " + hash)
		return c.String(http.StatusOK, "")
	})

	e.GET("/copyaspng2x/view", func(c echo.Context) error {
		time.After(5 * time.Second)
		hash := c.QueryParams().Get("hash")
		width := c.QueryParams().Get("width")

		var imageCh chan string
		exist := false

		defer func() error {
			ch, exist := imageMap[hash]
			if exist {
				close(ch)
				delete(imageMap, hash)
			}

			imageMap[hash] = make(chan string, 1)

			if !exist {
				return c.HTML(http.StatusOK, "<p>Try again.</p>")
			}

			return nil
		}()

		image := ""

		timeoutCh := time.After(5 * time.Second)
		imageCh, exist = imageMap[hash]
		if !exist {
			return nil
		}
		println("GET / " + c.RealIP() + " / " + hash)
		select {
		case <-timeoutCh:
			println("GET - Time out / ", c.RealIP(), " / ", hash)
			return c.HTML(http.StatusOK, "<p>Time out</p>")
		case image = <-imageCh:
		}

		return c.Render(http.StatusOK, "image.html", map[string]interface{}{
			"width": width,
			"image": image,
		})
	})

	certfile := "/etc/letsencrypt/live/figma.joowonyun.space/fullchain.pem"
	keyfile := "/etc/letsencrypt/live/figma.joowonyun.space/privkey.pem"
	e.TLSServer.Addr = ":443"
	graceful.ListenAndServeTLS(e.TLSServer, certfile, keyfile, 5*time.Second)
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
