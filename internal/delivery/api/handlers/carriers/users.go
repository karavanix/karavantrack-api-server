package carriers

import (
	"net/http"

	"github.com/go-chi/render"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/api/validation"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/outerr"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/users"
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

// SearchShippers godoc
// @Security     BearerAuth
// @Summary      Search shippers
// @Description  Search for shipper users by name, email, or phone
// @Tags         Users
// @Produce      json
// @Param        q query string true "Search query (name, email, or phone)"
// @Success      200  {array} query.UsersResponse
// @Failure      401  {object} outerr.Response
// @Failure      403  {object} outerr.Response
// @Router       /users/shippers/search [get]
func (h *usersHandler) SearchShippers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")

		resp, err := h.usersUsecase.Query.SearchShippers(r.Context(), &query.SearchShippersRequest{
			Query: q,
		})
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.JSON(w, r, resp)
	}
}
