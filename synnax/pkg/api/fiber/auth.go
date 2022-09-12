package fiber

import (
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/auth"
	"github.com/gofiber/fiber/v2"
)

type authService struct{ api.AuthService }

func (as *authService) Route(parent fiber.Router) {
	router := parent.Group("/auth")
	router.Post("/login", as.login)
	router.Post("/register", as.register)
	protected := router.Group("/protected")
	protected.Use(tokenMiddleware(as.Token))
	protected.Post("/change-password", as.changePassword)
	protected.Post("/change-username", as.changeUsername)
}

func (as *authService) login(c *fiber.Ctx) error {
	var creds auth.InsecureCredentials
	if err := parseBody(c, &creds); err.Occurred() {
		return errorResponse(c, err)
	}
	res, err := as.Login(creds)
	return maybeGoodResponse(c, err, res)
}

func (as *authService) register(c *fiber.Ctx) error {
	var req api.RegistrationRequest
	if err := parseBody(c, &req); err.Occurred() {
		return errorResponse(c, err)
	}
	res, err := as.Register(req)
	return maybeGoodResponse(c, err, res)
}

func (as *authService) changePassword(c *fiber.Ctx) error {
	var cpr api.ChangePasswordRequest
	if err := parseBody(c, &cpr); err.Occurred() {
		return errorResponse(c, err)
	}
	return maybeErrorResponse(c, as.ChangePassword(cpr))
}

func (as *authService) changeUsername(c *fiber.Ctx) error {
	var cpr api.ChangeUsernameRequest
	if err := parseBody(c, &cpr); err.Occurred() {
		return errorResponse(c, err)
	}
	return maybeErrorResponse(c, as.ChangeUsername(cpr))
}
