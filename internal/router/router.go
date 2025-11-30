package routes

import (
	_ "github.com/oggyb/insider-assessment/internal/docs" // swagger docs
	"github.com/oggyb/insider-assessment/internal/response"
	swaggerHandler "github.com/swaggo/http-swagger"
	"net/http"
)

type AppDeps struct {
	Home    HomeHandler
	Message MessageHandler
}

type HomeHandler interface {
	Index(w http.ResponseWriter, r *http.Request)
	Health(w http.ResponseWriter, r *http.Request)
}

type MessageHandler interface {
	GetSentMessages(w http.ResponseWriter, r *http.Request)
	StartStopScheduler(w http.ResponseWriter, r *http.Request)
}

func Register(mux *http.ServeMux, d AppDeps) {
	mux.HandleFunc("GET /{$}", d.Home.Index)
	mux.HandleFunc("GET /health", d.Home.Health)

	mux.HandleFunc("GET /messages/sent", d.Message.GetSentMessages)
	mux.HandleFunc("POST /scheduler", d.Message.StartStopScheduler)

	//Swagger
	mux.HandleFunc("GET /swagger/", swaggerHandler.WrapHandler)

	// Fallback handler for undefined routes (404)
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.RespondError(w, http.StatusNotFound, "route not found")
	}))
}
