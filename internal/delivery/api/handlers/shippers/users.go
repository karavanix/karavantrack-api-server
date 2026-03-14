package shippers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/render"
	"github.com/go-playground/form/v4"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/validation"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/outerr"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/users"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/users/command"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/users/query"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
)

type usersHandler struct {
	cfg          *config.Config
	validator    *validation.Validator
	usersUsecase *users.Usecase
}

func NewUsersHandler(opts *delivery.HandlerOptions) *usersHandler {
	return &usersHandler{
		cfg:          opts.Config,
		validator:    opts.Validator,
		usersUsecase: opts.UsersUsecase,
	}
}

// Invite godoc
// @Security     BearerAuth
// @Summary      Invite user
// @Description  Invite user
// @Tags         Users
// @Produce      json
// @Param        body body command.InviteRequest true "Invite user"
// @Success      200  {object} command.InviteResponse
// @Failure      401  {object} outerr.Response
// @Failure      403  {object} outerr.Response
// @Failure      404  {object} outerr.Response
// @Failure      500  {object} outerr.Response
// @Router       /users/invite [post]
func (h *usersHandler) Invite() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req command.InviteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			outerr.BadRequest(w, r, "invalid request body")
			return
		}
		if err := h.validator.Validate(req); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		var resp *command.InviteResponse
		resp, err := h.usersUsecase.Command.Invite(ctx, &req)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}

// GetCarrierByContact godoc
// @Security     BearerAuth
// @Summary      Get carrier by contact
// @Description  Get carrier user by contact
// @Tags         Users
// @Produce      json
// @Param        contact query string true "Email or Phone Number"
// @Success      200  {array} query.GetCarrierByContactResponse
// @Failure      401  {object} outerr.Response
// @Failure      403  {object} outerr.Response
// @Router       /users/carriers/by-contact [get]
func (h *usersHandler) GetCarrierByContact() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var urlForm query.GetCarrierByContactRequest
		if err := form.NewDecoder().Decode(&urlForm, r.URL.Query()); err != nil {
			outerr.BadRequest(w, r, "invalid request form: "+err.Error())
			return
		}

		if err := h.validator.Validate(urlForm); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		var resp *query.GetCarrierByContactResponse
		resp, err := h.usersUsecase.Query.GetCarrierByContact(ctx, &urlForm)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}

// GetShipperByContact godoc
// @Security     BearerAuth
// @Summary      Get shipper by contact
// @Description  Get shipper user by contact
// @Tags         Users
// @Produce      json
// @Param        contact query string true "Email or Phone Number"
// @Success      200  {array} query.GetShipperByContactResponse
// @Failure      401  {object} outerr.Response
// @Failure      403  {object} outerr.Response
// @Router       /users/shippers/by-contact [get]
func (h *usersHandler) GetShipperByContact() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var urlForm query.GetShipperByContactRequest
		if err := form.NewDecoder().Decode(&urlForm, r.URL.Query()); err != nil {
			outerr.BadRequest(w, r, "invalid request form: "+err.Error())
			return
		}

		if err := h.validator.Validate(urlForm); err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		var resp *query.GetShipperByContactResponse
		resp, err := h.usersUsecase.Query.GetShipperByContact(ctx, &urlForm)
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}
