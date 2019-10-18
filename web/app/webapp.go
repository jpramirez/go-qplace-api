package app

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	models "github.com/jpramirez/go-qplace-api/pkg/models"
)

type JResponse struct {
	ResponseCode string
	Message      string
	ResponseData []byte
}

type JResponseFileStatus struct {
	ResponseCode string
	Message      string
	FileStatus   []ResponseFileStatus
}

type ResponseFileStatus struct {
	FileName string
	Status   string
	Hash     string
}

//MainWebApp PHASE
type MainWebApp struct {
	Mux          *mux.Router
	Log          *log.Logger
	Config       models.Config
	ServerConfig models.ServerConfig
	Store        *sessions.CookieStore
}

//GetFileContentType will get the mime type of the file by reading its first 512 bytes (according to the standard)
func GetFileContentType(buffer []byte) (string, error) {
	// Use the net/http package's handy DectectContentType function. Always returns a valid
	// content-type by returning "application/octet-stream" if no others seemed to match.
	contentType := http.DetectContentType(buffer)
	return contentType, nil
}

//NewApp creates a new instances
func NewApp(config models.Config) (MainWebApp, error) {

	var err error
	var wapp MainWebApp
	mux := mux.NewRouter().StrictSlash(true)
	f, err := os.OpenFile(config.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)
	log := log.New(os.Stdout, "web ", log.LstdFlags)

	wapp.Mux = mux
	wapp.Config = config
	wapp.Log = log
	wapp.Store = sessions.NewCookieStore([]byte("7b24afc8bc80e548d66c4e7ff72171c5"))

	//wapp.Client, err = wapp.connectRPC()

	log.Println("NewAPP ---> Loggig Location")
	return wapp, err
}

func (M *MainWebApp) getSession(w http.ResponseWriter, r *http.Request) *sessions.Session {

	session, err := M.Store.Get(r, M.Config.AppName)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return nil
	}
	return session
}

func (M *MainWebApp) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	M.Mux.ServeHTTP(w, r)
}

func renderError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(message))
}

//UploadHandler is in charge of optimizing upload request
func (M *MainWebApp) UploadHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	allowedContent := M.Config.AudioFormats
	reader, err := r.MultipartReader()

	fmt.Println(err)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var response JResponseFileStatus

	for {
		var payload models.Payload
		var filestatus ResponseFileStatus

		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		//if part.FileName() is empty, skip this iteration.
		if part.FileName() == "" {
			continue
		}

		filestatus.FileName = part.FileName()
		fmt.Println("Processing file")

		dst, err := os.Create(M.Config.UploadFolder + filestatus.FileName)
		defer dst.Close()
		if err != nil {
			fmt.Println("Error opening file")

			//http.Error(w, err.Error(), http.StatusInternalServerError)
			//Error Opening the file skip iteration
			continue
		}
		buffer := make([]byte, 512)
		part.Read(buffer)
		//Here we send only 512 bytes to detect the right type
		fmt.Println("Processing content type")

		content, err := GetFileContentType(buffer)
		allowed := false
		for _, i := range allowedContent {
			fmt.Println(i)

			if i == content {
				fmt.Println("Allowed Content")
				allowed = true
				break
			}
		}

		fmt.Println("Processing allowed ?", content)

		if !allowed {
			continue
		}

		payload, err = models.NewPayLoad(part.FileName(), content)
		payload.StorageFolder = M.Config.UploadFolder
		var read int64
		var p float32

		length := r.ContentLength
		ticker := time.Tick(time.Millisecond) // <-- use this in production
		//ticker := time.Tick(time.Second) // this is for demo purpose with longer delay
		fmt.Println("Processing file uploading starting")

		for {

			buffer := make([]byte, 100000)
			cBytes, err := part.Read(buffer)
			if err == io.EOF {
				fmt.Printf("\n Last buffer read!")
				fmt.Println("\n Saving Payload ID", payload.PayloadID)
				dst.Close()
				payload.CalculateHASH()
				//_ = payload.Recongnize()
				filestatus.Hash = payload.FileHash
				filestatus.Status = "Uploaded"
				response.FileStatus = append(response.FileStatus, filestatus)
				break
			}
			read = read + int64(cBytes)

			if read > 0 {
				p = float32(read*100) / float32(length)
				//fmt.Printf("progress: %v \n", p)
				<-ticker
				fmt.Printf("\rUploading progress %v", p) // for console
				dst.Write(buffer[0:cBytes])
			} else {
				break
			}

		}

	}

	response.ResponseCode = "200"
	response.Message = "File Uploaded"
	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "Application/json")
	w.Write(js)
}

//Liveness just keeps the connection alive
func (M *MainWebApp) Liveness(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	//if a.CheckSession(w, r) == false {
	//	http.Error(w, "unauthorised", http.StatusUnauthorized)
	//	return
	//}
	var response JResponse

	response.ResponseCode = "200 OK"
	response.Message = "alive"
	response.ResponseData = []byte("")
	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "Application/json")
	w.Write(js)
}
