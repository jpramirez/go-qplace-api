package web

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	models "github.com/jpramirez/go-qplace-api/pkg/models"
	webapp "github.com/jpramirez/go-qplace-api/web/app"
)

//WebAgent is the main struct for this agent.
type WebOne struct {
	webConfig models.Config
}

//StartServer Starts the server using the variable sip and port, creates anew instance.
func (W *WebOne) StartServer() {

	handler := W.New()

	srv := &http.Server{
		Handler:      handlers.CORS(handlers.AllowedOrigins([]string{"*"}))(handlers.LoggingHandler(os.Stdout, handler)),
		Addr:         W.webConfig.WebAddress + ":" + W.webConfig.WebPort,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	err := srv.ListenAndServeTLS(W.webConfig.CrtFile, W.webConfig.KeyFile)

	if err != nil {
		log.Println("Error Starting web server")
	}
}

//NewWebAgent creates new instance.
func NewWebAgent(config models.Config, BuildVersion string, BuidTime string) (WebOne, error) {
	var webone WebOne
	log.Println("Starting Go Gif Go")
	log.Println("Version : " + BuildVersion)
	log.Println("Build Time : " + BuidTime)

	webone.webConfig = config
	log.Println("Listening on ", webone.webConfig.WebAddress, webone.webConfig.WebPort)

	// Stop the grpc verbose logging
	//grpclog.SetLogger(noplog)
	return webone, nil
}

//New creates a new handler
func (W *WebOne) New() http.Handler {

	app, err := webapp.NewApp(W.webConfig)

	if err != nil {
		log.Fatalln("Error creating WebApp", err)
		return nil
	}
	api := app.Mux.PathPrefix("/api/v1").Subrouter()
	// API Calls
	api.HandleFunc("/upload", app.UploadHandler)
	api.HandleFunc("/liveness", app.Liveness)
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ch
		log.Println("Closing system")
		os.Exit(0)
	}()
	return &app
}
