package main

import (
	"io"
	"sync"
	"net/http"
	"html/template"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
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

	var mutex = new(sync.Mutex)
	var cond = sync.NewCond(mutex)

	imageMap := make(map[string]chan string)

	e.POST("/copyaspng2x/image", func(c echo.Context) error {
		hash := c.FormValue("hash")
		image := c.FormValue("image")

		ch := make(chan string, 1)

		cond.L.Lock()
		imageMap[hash] = ch
		imageMap[hash] <-image
		cond.Broadcast()
		cond.L.Unlock()

		println("POST / " + c.RealIP() + " / " + hash)
		return c.String(http.StatusOK, "")
	})

	e.GET("/copyaspng2x/view", func(c echo.Context) error {
		hash := c.QueryParams().Get("hash")
		width := c.QueryParams().Get("width")
		
		var imageCh chan string
		exist := false
		cond.L.Lock()
		for true {
			imageCh, exist = imageMap[hash]
			if exist {
				break
			}
			cond.Wait() 
		}
		cond.L.Unlock()
		image := <-imageCh

		println("GET / " + c.RealIP() + " / " + hash)
		if !exist {
			return c.HTML(http.StatusOK, "<p>Try again.</p>")
		}

		delete(imageMap, hash)

		return c.Render(http.StatusOK, "image.html", map[string]interface{}{
			"width": width,
			"image": image,
		})
	})

	certfile := "/etc/letsencrypt/live/figma.joowonyun.space/fullchain.pem"
	keyfile := "/etc/letsencrypt/live/figma.joowonyun.space/privkey.pem"
	e.Logger.Fatal(e.StartTLS(":443", certfile, keyfile))
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