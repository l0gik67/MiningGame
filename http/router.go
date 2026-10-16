package http

import (
	"net/http"

	"github.com/gorilla/mux"
)

func NewGameRouter() http.Handler {
	return NewGameHandler().Router()
}

func (h *GameHandler) Router() http.Handler {
	router := mux.NewRouter()

	router.HandleFunc("/miners/types", h.getAllMinerTypes).Methods(http.MethodGet)
	router.HandleFunc("/miners", h.getMiners).Methods(http.MethodGet)
	router.HandleFunc("/miners", h.hireNewMiner).Methods(http.MethodPost)

	router.HandleFunc("/equipment", h.getEquipment).Methods(http.MethodGet)
	router.HandleFunc("/equipment", h.buyEquipment).Methods(http.MethodPost)

	router.HandleFunc("/enterprise", h.getEnterpriseInfo).Methods(http.MethodGet)
	router.HandleFunc("/enterprise/completion", h.completeGame).Methods(http.MethodPost)
	router.HandleFunc("/enterprise/complition", h.completeGame).Methods(http.MethodPost)
	router.HandleFunc("/enterprise/beginning", h.startGame).Methods(http.MethodPost)
	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "Маршрут не найден")
	})
	router.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
	})

	return router
}
