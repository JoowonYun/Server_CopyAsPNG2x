package main

import (
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

func main() {
	e := echo.New()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
	}))

	imageMap := make(map[string]string)

	e.POST("/copyaspng2x/image", func(c echo.Context) error {
		hash := c.FormValue("hash")
		image := c.FormValue("image")

		imageMap[hash] = image

		println("POST / " + c.RealIP() + " / " + hash)
		return c.String(http.StatusOK, "")
	})

	e.GET("/copyaspng2x/view", func(c echo.Context) error {
		hash := c.QueryParams().Get("hash")
		width := c.QueryParams().Get("width")
		image, exist := imageMap[hash]

		println("GET / " + c.RealIP() + " / " + hash)
		if !exist {
			return c.HTML(http.StatusOK, "<p>Try again.</p>")
		}

		delete(imageMap, hash)

		return c.HTML(http.StatusOK, `
		<style>
			body {
				margin: 0;
			}
			.wrap {
				display: flex;
				justify-content: center;
				flex-wrap: wrap;
			}
			img {
				width: `+width+`px;
			}
		</style>

		<div class="wrap">
			<img id="base64img" src=`+`data:image/png;base64,`+ image +`>
		</div>
		<script type="text/javascript">
		var imgElm = document.getElementById("base64img")

		// for copy
		var canvas = document.createElement("canvas")
		canvas.width = imgElm.clientWidth;
		canvas.height = imgElm.clientHeight;
  
		let context = canvas.getContext('2d');
  
		context.drawImage(imgElm, 0, 0);
		
		try {
			canvas.toBlob((blob) => { 
			  console.log(blob)
			  const item = new ClipboardItem({"image/png": blob });
			  navigator.clipboard.write([item]); 
			});
		  } catch (error) {
			console.log(error)
		  }
		
	  </script>`)
	})

	certfile := "/etc/letsencrypt/live/figma.joowonyun.space/fullchain.pem"
	keyfile := "/etc/letsencrypt/live/figma.joowonyun.space/privkey.pem"
	e.Logger.Fatal(e.StartTLS(":443", certfile, keyfile))
}
