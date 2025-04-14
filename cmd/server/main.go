package main

import (
	"football-server/internal/database"
	"football-server/internal/handlers"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	db := database.Connect()
	defer db.Close()

	matchHandler := handlers.NewMatchHandler(db)

	r := chi.NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/templates/index.html")
	})
	r.Get("/league", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/templates/league.html")
	})
	r.Route("/api", func(r chi.Router) {
		r.HandleFunc("/leagues", matchHandler.GetLeagues)
		r.HandleFunc("/top-teams", matchHandler.GetTopTeams)
		r.HandleFunc("/top-players", matchHandler.GetTopPlayers)
		r.HandleFunc("/matches", matchHandler.GetMatches)
		r.HandleFunc("/seasons", matchHandler.GetSeasons)
	})

	r.Handle("/*", http.StripPrefix("/", http.FileServer(http.Dir("./web"))))
	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", r))
}
