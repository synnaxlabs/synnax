package fiber

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"io/fs"
	"net/http"
)

type uiService struct {
	Dist fs.FS
}

func (us *uiService) Route(router fiber.Router) {
	router.Use("/", filesystem.New(filesystem.Config{
		Root:       http.FS(us.Dist),
		PathPrefix: "dist",
		Browse:     true,
	}))
}
