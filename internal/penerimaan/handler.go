package penerimaan

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/minouse3/sistem-apotek/internal/auth"
	"github.com/minouse3/sistem-apotek/internal/middleware"
	"github.com/minouse3/sistem-apotek/internal/view"
)

type pageRenderer interface {
	Render(w http.ResponseWriter, status int, page string, data any)
}

type Handler struct {
	pbf       http.Handler
	receiving *ReceivingHandler
}

func NewHandler(pool *pgxpool.Pool, renderer *view.Renderer) *Handler {
	return &Handler{
		pbf:       newHandler(postgresDistributorStore{pool: pool}, renderer),
		receiving: &ReceivingHandler{handler: newReceivingHandler(postgresReceivingStore{pool: pool}, renderer)},
	}
}

func (h *Handler) Register(mux *http.ServeMux, accounts *auth.Service) {
	distributorRoutes := middleware.Require(accounts, h.pbf, auth.RoleStaf, auth.RolePemilik)
	mux.Handle("/pbf", distributorRoutes)
	mux.Handle("/pbf/", distributorRoutes)

	receivingRoutes := middleware.Require(accounts, h.receiving.Routes(), auth.RoleStaf, auth.RolePemilik)
	mux.Handle("/penerimaan", receivingRoutes)
	mux.Handle("/penerimaan/", receivingRoutes)

	mux.Handle(
		"POST /penerimaan/{id}/cancel",
		middleware.Require(accounts, h.receiving.CancelHandler(), auth.RolePemilik),
	)
}
