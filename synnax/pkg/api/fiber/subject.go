package fiber

import (
	apierrors "github.com/synnaxlabs/synnax/pkg/api/errors"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/cockroachdb/errors"
	"github.com/gofiber/fiber/v2"
)

const subjectKey = "subject"

func setSubject(c *fiber.Ctx, key ontology.ID) { c.Locals(subjectKey, key) }

// getSubject retrieves the subject of a request (the entity attempting to perform
// an action on an object). Returns false if the subject is not set on the request.
func getSubject(c *fiber.Ctx) (ontology.ID, apierrors.Typed) {
	key, ok := c.Locals(subjectKey).(ontology.ID)
	if !ok {
		return key, apierrors.Unexpected(errors.New(
			"[access] - subject not set on query. this is a bug.",
		))
	}
	return key, apierrors.Nil
}
