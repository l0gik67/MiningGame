package http

import (
	"net/http"

	"github.com/gorilla/mux"
)

func NewGameRouter() http.Handler {
	router := mux.NewRouter()

	router.HandleFunc("/miners/types", getAllMinerTypes).Methods(http.MethodGet)
	router.HandleFunc("/miners", getMiners).Methods(http.MethodGet)
	router.HandleFunc("/miners", hireNewMiner).Methods(http.MethodPost)

	router.HandleFunc("/equipment", getEquipment).Methods(http.MethodGet)
	router.HandleFunc("/equipment", buyEquipment).Methods(http.MethodPost)

	router.HandleFunc("/enterprise", getEnterpriseInfo).Methods(http.MethodGet)
	router.HandleFunc("/enterprise/complition", completeGame).Methods(http.MethodPost)
	router.HandleFunc("/enterprise/beginning", startGame).Methods(http.MethodPost)

	return router
}
