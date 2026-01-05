package admin

import (
	"github.com/gofiber/fiber/v2"
	adminmw "github.com/qolzam/telar/apps/api/internal/middleware/admin"
	"github.com/qolzam/telar/apps/api/admin/members"
	"github.com/qolzam/telar/apps/api/admin/moderation"
)

type Handlers struct {
	Members    *members.Handler
	Moderation *moderation.Handler
}

type RouterConfig struct {
}

func RegisterRoutes(app *fiber.App, handlers *Handlers) {
	group := app.Group("/admin",
		adminmw.New(adminmw.Config{}),
	)

	membersHandler := handlers.Members
	membersGroup := group.Group("/members")
	membersGroup.Get("/", membersHandler.List)
	membersGroup.Get("/:userId", membersHandler.Get)
	membersGroup.Put("/:userId/role", membersHandler.UpdateRole)
	membersGroup.Post("/:userId/ban", membersHandler.Ban)

	// Moderation Group
	modHandler := handlers.Moderation
	modGroup := group.Group("/moderation")
	modGroup.Get("/queue", modHandler.GetQueue)
	modGroup.Post("/:id/approve", modHandler.Approve)
	modGroup.Post("/:id/reject", modHandler.Reject)
}






