package members

import (
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	uuid "github.com/gofrs/uuid"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) List(c *fiber.Ctx) error {
	limit := 20
	offset := 0
	search := c.Query("search", "")
	sortBy := c.Query("sortBy", "created_date")
	sortOrder := c.Query("sortOrder", "desc")
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	if o := c.Query("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}
	params := ListMembersParams{
		Search:    search,
		Limit:     limit,
		Offset:    offset,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}
	result, err := h.svc.List(c.Context(), params)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list members"})
	}
	return c.JSON(result)
}

func (h *Handler) Get(c *fiber.Ctx) error {
	idStr := c.Params("userId")
	id, err := uuid.FromString(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid userId"})
	}
	item, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "member not found"})
	}
	return c.JSON(item)
}

func (h *Handler) UpdateRole(c *fiber.Ctx) error {
	idStr := c.Params("userId")
	id, err := uuid.FromString(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid userId"})
	}
	var body struct {
		Role string `json:"role"`
	}
	if err := c.BodyParser(&body); err != nil || body.Role == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "role is required"})
	}
	if err := h.svc.UpdateRole(c.Context(), id, body.Role); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update role"})
	}
	return c.Status(http.StatusOK).JSON(fiber.Map{"message": "role updated"})
}

func (h *Handler) Ban(c *fiber.Ctx) error {
	idStr := c.Params("userId")
	id, err := uuid.FromString(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid userId"})
	}
	if err := h.svc.Ban(c.Context(), id); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to ban member"})
	}
	return c.Status(http.StatusOK).JSON(fiber.Map{"message": "member banned"})
}


