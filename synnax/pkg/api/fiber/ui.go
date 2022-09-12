package fiber

import (
	"bytes"
	"github.com/synnaxlabs/synnax/pkg/ui"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"io/fs"
	"net/http"
)

type uiService struct {
	HaveUI bool
	Dist   fs.FS
}

func (us *uiService) Route(router fiber.Router) {
	if us.HaveUI {
		router.Use("/", filesystem.New(filesystem.Config{
			Root:       http.FS(us.Dist),
			PathPrefix: "dist",
			Browse:     true,
		}))
	} else {
		router.Get("/", func(c *fiber.Ctx) error {
			return c.SendStream(bytes.NewReader(ui.BareHTML))
		})
	}

}
