package common

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/render"
	"github.com/karavanix/karavantrack-api-server/internal/delivery"
	"github.com/karavanix/karavantrack-api-server/internal/delivery/outerr"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/leads"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/leads/command"
)

type publicLeadsHandler struct {
	leadsUsecase *leads.Usecase
}

func NewPublicLeadsHandler(opts *delivery.HandlerOptions) *publicLeadsHandler {
	return &publicLeadsHandler{leadsUsecase: opts.LeadsUsecase}
}

type submitLeadRequest struct {
	Name     string `json:"name"`
	Company  string `json:"company"`
	Phone    string `json:"phone"`
	Fleet    string `json:"fleet"`
	Honeypot string `json:"honeypot"`
}

// SubmitLead godoc
// @Summary      Submit sales lead
// @Description  PUBLIC, unauthenticated contact form submission from the marketing landing page.
// @Tags         Leads
// @Accept       json
// @Produce      json
// @Param        body body submitLeadRequest true "Lead form data"
// @Success      200
// @Failure      400  {object} outerr.Response
// @Router       /leads [post]
func (h *publicLeadsHandler) SubmitLead() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req submitLeadRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			outerr.BadRequest(w, r, "invalid request body")
			return
		}

		// Bot honeypot: silently accept without writing anything.
		if req.Honeypot != "" {
			render.Status(r, http.StatusOK)
			render.JSON(w, r, map[string]bool{"ok": true})
			return
		}

		err := h.leadsUsecase.Command.SubmitLead(r.Context(), command.SubmitLeadRequest{
			Name:      req.Name,
			Company:   req.Company,
			Phone:     req.Phone,
			Fleet:     req.Fleet,
			UserAgent: r.Header.Get("User-Agent"),
		})
		if err != nil {
			outerr.HandleHTTP(w, r, err)
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, map[string]bool{"ok": true})
	}
}
