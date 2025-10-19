package profile

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofrs/uuid"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/qolzam/telar/apps/api/profile/errors"
	"github.com/qolzam/telar/apps/api/profile/models"
	"github.com/qolzam/telar/apps/api/profile/services"
	"github.com/qolzam/telar/apps/api/profile/validation"
)

type ProfileHandler struct {
	profileService *services.Service
	jwtConfig      platformconfig.JWTConfig
	hmacConfig     platformconfig.HMACConfig
}

func NewProfileHandler(profileService *services.Service, jwtConfig platformconfig.JWTConfig, hmacConfig platformconfig.HMACConfig) *ProfileHandler {
	return &ProfileHandler{
		profileService: profileService,
		jwtConfig:      jwtConfig,
		hmacConfig:     hmacConfig,
	}
}

func (h *ProfileHandler) ReadMyProfile(c *fiber.Ctx) error {
	if uc, ok := c.Locals(types.UserCtxName).(types.UserContext); ok && uc.UserID != uuid.Nil {
		doc, err := h.profileService.FindMyProfile(c.Context(), uc.UserID)
		if err != nil {
			return errors.HandleServiceError(c, err)
		}
		if doc == nil {
			return errors.HandleNotFoundError(c, "Profile not found")
		}
		return c.JSON(doc)
	}
	return errors.HandleUnauthorizedError(c, "Authentication required")
}

func (h *ProfileHandler) QueryUserProfile(c *fiber.Ctx) error {
	search := c.Query("search", "")

	pageStr := c.Query("page", "1")
	page, err := strconv.ParseInt(pageStr, 10, 64)
	if err != nil || page <= 0 {
		page = 1
	}

	limitStr := c.Query("limit", "10")
	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil || limit <= 0 {
		limit = 10
	}

	filter := &models.ProfileQueryFilter{
		Search: search,
		Page:   page,
		Limit:  limit,
	}

	if err := validation.ValidateProfileQueryFilter(filter); err != nil {
		return errors.HandleValidationError(c, err.Error())
	}

	docs, err := h.profileService.Query(c.Context(), filter.Search, filter.Limit, (filter.Page-1)*filter.Limit)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}
	if docs == nil {
		docs = []*models.Profile{}
	}
	return c.JSON(docs)
}

func (h *ProfileHandler) ReadProfile(c *fiber.Ctx) error {
	idStr := c.Params("userId")
	id, err := uuid.FromString(idStr)
	if err != nil {
		return errors.HandleUUIDError(c, "userId")
	}
	doc, err := h.profileService.FindByID(c.Context(), id)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}
	return c.JSON(doc)
}

func (h *ProfileHandler) GetBySocialName(c *fiber.Ctx) error {
	name := c.Params("name")

	if err := validation.ValidateSocialName(name); err != nil {
		return errors.HandleValidationError(c, err.Error())
	}

	doc, err := h.profileService.FindBySocialName(c.Context(), name)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}
	return c.JSON(doc)
}

func (h *ProfileHandler) GetProfileByIds(c *fiber.Ctx) error {
	var idsStr []string
	if err := json.Unmarshal(c.Body(), &idsStr); err != nil {
		return errors.HandleInvalidRequestError(c, "Invalid JSON body - expected array of UUID strings")
	}

	if err := validation.ValidateUserIdList(idsStr); err != nil {
		return errors.HandleValidationError(c, err.Error())
	}

	ids := make([]uuid.UUID, 0, len(idsStr))
	for _, s := range idsStr {
		if id, err := uuid.FromString(s); err == nil {
			ids = append(ids, id)
		}
	}
	docs, err := h.profileService.FindManyByIDs(c.Context(), ids)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}
	if docs == nil {
		docs = []*models.Profile{}
	}
	return c.JSON(docs)
}

func (h *ProfileHandler) InitProfileIndex(c *fiber.Ctx) error {
	if err := h.profileService.CreateIndexes(c.Context()); err != nil {
		return errors.HandleServiceError(c, err)
	}
	return c.SendStatus(http.StatusOK)
}

func (h *ProfileHandler) UpdateLastSeen(c *fiber.Ctx) error {
	var req models.UpdateLastSeenRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return errors.HandleInvalidRequestError(c, "Invalid JSON body")
	}

	if err := validation.ValidateUpdateLastSeenRequest(&req); err != nil {
		return errors.HandleValidationError(c, err.Error())
	}

	userId, err := uuid.FromString(req.UserId)
	if err != nil {
		return errors.HandleUUIDError(c, "userId")
	}
	if err := h.profileService.UpdateLastSeen(c.Context(), userId, 1); err != nil {
		return errors.HandleServiceError(c, err)
	}
	return c.SendStatus(http.StatusOK)
}

func (h *ProfileHandler) UpdateProfile(c *fiber.Ctx) error {
	var req models.UpdateProfileRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return errors.HandleInvalidRequestError(c, "Invalid JSON body")
	}

	if err := validation.ValidateUpdateProfileRequest(&req); err != nil {
		return errors.HandleValidationError(c, err.Error())
	}

	if uc, ok := c.Locals(types.UserCtxName).(types.UserContext); ok && uc.UserID != uuid.Nil {
		if err := h.profileService.UpdateProfile(c.Context(), uc.UserID, &req); err != nil {
			return errors.HandleServiceError(c, err)
		}
		return c.SendStatus(http.StatusOK)
	}
	return errors.HandleUnauthorizedError(c, "Authentication required")
}

func (h *ProfileHandler) ReadDtoProfile(c *fiber.Ctx) error {
	idStr := c.Params("userId")
	id, err := uuid.FromString(idStr)
	if err != nil {
		return errors.HandleUUIDError(c, "userId")
	}
	doc, err := h.profileService.FindByID(c.Context(), id)
	if err != nil {
		return errors.HandleServiceError(c, err)
	}
	if doc == nil {
		return errors.HandleNotFoundError(c, "Profile not found")
	}
	return c.JSON(doc)
}

func (h *ProfileHandler) CreateDtoProfile(c *fiber.Ctx) error {
	var req models.CreateProfileRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return errors.HandleInvalidRequestError(c, "Invalid JSON body")
	}

	if err := validation.ValidateCreateProfileRequest(&req); err != nil {
		return errors.HandleValidationError(c, err.Error())
	}

	if err := h.profileService.CreateOrUpdateDTO(c.Context(), &req); err != nil {
		return errors.HandleServiceError(c, err)
	}
	return c.SendStatus(http.StatusCreated)
}

func (h *ProfileHandler) DispatchProfiles(c *fiber.Ctx) error {
	var payload map[string]interface{}
	if err := json.Unmarshal(c.Body(), &payload); err != nil {
		return errors.HandleInvalidRequestError(c, "Invalid JSON body")
	}
	// No-op endpoint for compatibility
	return c.SendStatus(http.StatusOK)
}

func (h *ProfileHandler) IncreaseFollowCount(c *fiber.Ctx) error {
	id, err := uuid.FromString(c.Params("userId"))
	if err != nil {
		return errors.HandleUUIDError(c, "userId")
	}
	inc, err := strconv.Atoi(c.Params("inc"))
	if err != nil {
		return errors.HandleInvalidFieldError(c, "inc", "invalid increment value")
	}

	if err := validation.ValidateIncrementValue(inc); err != nil {
		return errors.HandleValidationError(c, err.Error())
	}

	if err := h.profileService.Increase(c.Context(), "followCount", inc, id); err != nil {
		return errors.HandleServiceError(c, err)
	}
	return c.SendStatus(http.StatusOK)
}

func (h *ProfileHandler) IncreaseFollowerCount(c *fiber.Ctx) error {
	id, err := uuid.FromString(c.Params("userId"))
	if err != nil {
		return errors.HandleUUIDError(c, "userId")
	}
	inc, err := strconv.Atoi(c.Params("inc"))
	if err != nil {
		return errors.HandleInvalidFieldError(c, "inc", "invalid increment value")
	}

	if err := validation.ValidateIncrementValue(inc); err != nil {
		return errors.HandleValidationError(c, err.Error())
	}

	if err := h.profileService.Increase(c.Context(), "followerCount", inc, id); err != nil {
		return errors.HandleServiceError(c, err)
	}
	return c.SendStatus(http.StatusOK)
}

