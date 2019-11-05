package app

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"html/template"
	"strings"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	uuid "github.com/satori/go.uuid"

	models "github.com/jpramirez/go-qplace-api/pkg/models"
	storage "github.com/jpramirez/go-qplace-api/pkg/storage"
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
	storage      *storage.Client
}

//GetFileContentType will get the mime type of the file by reading its first 512 bytes (according to the standard)
func GetFileContentType(buffer []byte) (string, error) {
	// Use the net/http package's handy DectectContentType function. Always returns a valid
	// content-type by returning "application/octet-stream" if no others seemed to match.
	contentType := http.DetectContentType(buffer)
	return contentType, nil
}

//NewApp creates a new instances
func NewApp(config models.Config, db *storage.Client) (MainWebApp, error) {

	var err error
	var wapp MainWebApp

	wapp.storage = db
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

	session, err := M.Store.Get(r, "qplace-go-session")
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

//V1GetAllFiles will get all files for the logged user
func (M *MainWebApp) V1GetAllFiles(w http.ResponseWriter, r *http.Request) {

	setupResponse(&w, r)

	if r.Method == "OPTIONS" {
		return
	}

	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)

	userID := vars["userid"]
	fmt.Println("userID ", userID)

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

//Liveness just keeps the connection alive
func (M *MainWebApp) Liveness(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if M.CheckSession(w, r) == false {
		http.Error(w, "unauthorised", http.StatusUnauthorized)
		return
	}
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

// Auth Credentials API Calls

//V1Login main login function to keep also store
func (M *MainWebApp) V1Login(w http.ResponseWriter, r *http.Request) {
	log.Println("Getting response before options")

	setupResponse(&w, r)

	log.Println("Getting response before options")
	if r.Method == "OPTIONS" {
		return
	}
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var err error

	session, err := M.Store.Get(r, "qplace-go-session")

	var _user models.JSONLogin
	var user models.User
	err = json.NewDecoder(r.Body).Decode(&_user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Println("User ", _user.Email)

	user.Email = _user.Email
	user.Password = []byte(_user.Password)

	user, auth, err := M.storage.CheckUser(user)

	if err != nil {
		/*
			var response JResponse
			response.ResponseCode = "201"
			response.Message = "incorrect Username or Password "
			response.ResponseData = []byte("")
			jresponse, err := json.Marshal(response)
		*/
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "Application/json")
		//w.Write(jresponse)
		http.Error(w, "Not authorized", 401)

		return
	}
	if auth {
		user.Password = []byte("") // We empty the password
		var usersResponse []models.User
		var response models.JSONResponseUsers

		usersResponse = append(usersResponse, user)
		response.ResponseCode = "200"
		response.Message = "logged in Succesfully"
		response.ResponseData = usersResponse

		jresponse, err := json.Marshal(response)

		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "Application/json")
		session.Options.Path = "/"
		session.Options.MaxAge = 3600
		session.Options.HttpOnly = true
		session.Values["user"] = user.UserID
		err = session.Save(r, w)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write(jresponse)
		return
	} else {

		/*
			var response JResponse
			response.ResponseCode = "201"
			response.Message = "incorrect Username or Password "
			response.ResponseData = []byte("")
			jresponse, err := json.Marshal(response)

			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "Application/json")
			w.Write(jresponse)
		*/
		http.Error(w, err.Error(), http.StatusForbidden)

		return
	}

}

//V1Logout destro session
func (M *MainWebApp) V1Logout(w http.ResponseWriter, r *http.Request) {
	setupResponse(&w, r)
	if r.Method != "GET" {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	session := M.getSession(w, r)
	w.Header().Set("Content-Type", "Application/json")
	session.Options.Path = "/"
	session.Options.MaxAge = -1 //We remove the session completely
	session.Options.HttpOnly = true
	M.Store.Save(r, w, session)
	w.Header().Set("Content-Type", "Application/json")
	http.Redirect(w, r, "/", 301)

}

//CheckSession validates that user has a session active
func (M *MainWebApp) CheckSession(w http.ResponseWriter, r *http.Request) bool {
	setupResponse(&w, r)

	session := M.getSession(w, r)
	// MOCK function we should add server status , this is a TEST WIP TODO session

	userID, found := session.Values["user"]
	if !found {
		fmt.Println("No user_id found in session")
		return false
	}

	str := fmt.Sprintf("%v", userID)
	user, err := M.storage.CheckUserByID(str)

	if err != nil {
		log.Println("Session Failed to renew or Expired")
		http.Error(w, "unauthorised", http.StatusUnauthorized)
		return false
	}
	if user.Email == "" {
		log.Println("Session Failed to renew or Expired")
		http.Error(w, "unauthorised", http.StatusUnauthorized)
		return false
	}

	M.Store.MaxAge(3600) // renew session 1 Minute
	M.Store.Save(r, w, session)
	return true
}

//V1CreateUser destro session
func (M *MainWebApp) V1CreateUser(w http.ResponseWriter, r *http.Request) {
	setupResponse(&w, r)

	if r.Method == "OPTIONS" {
		return
	}
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var response JResponse
	log.Printf(r.Header.Get("AdminToken"))
	log.Println(r.Header)

	_token := r.Header.Get("AdminToken")

	if _token != "" {
		token, err := M.storage.CheckToken(_token)
		fmt.Println(token != (models.Token{}))
		fmt.Println(err)

		if err != nil {
			fmt.Println("Error ", err)
		}
		if token != (models.Token{}) {
			if token.IsAdmin {

				var _user models.JSONCreateUser

				err = json.NewDecoder(r.Body).Decode(&_user)

				var tempUser models.User
				// or error handling
				u2, err := uuid.NewV4()
				if err != nil {
					response.ResponseCode = "401"
					response.Message = "Error Creating user "
					response.ResponseData = []byte("")
				}
				tempUser.Username = _user.UserName
				tempUser.Email = _user.Email
				tempUser.Password = []byte(_user.Password) //Default Password CHANGE IN PROD
				tempUser.UserID = u2.String()
				tempUser.Token = ""
				tempUser.Approved = false
				tempUser.Banned = true
				tempUser.Role = "Admin"
				err = M.storage.UserAdd(tempUser)
				if err != nil {
					response.ResponseCode = "401"
					response.Message = "Error Creating user "
					response.ResponseData = []byte("")
				}
				response.ResponseCode = "201"
				response.Message = "User Created"
				response.ResponseData = []byte("")
			} else {
				response.ResponseCode = "401"
				response.Message = "Token is not admin"
				response.ResponseData = []byte("")
			}
		} else {
			response.ResponseCode = "401"
			response.Message = "Error Creating user, token not found "
			response.ResponseData = []byte("")
		}

	} else {
		response.ResponseCode = "201"
		response.Message = "Error Creating user "
		response.ResponseData = []byte("")
	}
	jresponse, err := json.Marshal(response)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "Application/json")
	if response.ResponseCode == "401" {
		http.Error(w, "Not authorized", 401)

	} else {
		w.Write(jresponse)
	}

	return

}

//HandleIndex for serving SPA
func (M *MainWebApp) HandleIndex(w http.ResponseWriter, r *http.Request) {

	t := template.Must(template.New("").ParseGlob("templates/*.html"))
	t.ExecuteTemplate(w, "index.html", map[string]interface{}{
		csrf.TemplateTag: csrf.TemplateField(r),
		"Stage":          os.Getenv("UP_STAGE"),
		"Year":           time.Now().Format("2006"),
		"EmojiCountry":   countryFlag(strings.Trim(r.Header.Get("Cloudfront-Viewer-Country"), "[]")),
	})
}

//UploadHandler is in charge of optimizing upload request
func (M *MainWebApp) UploadHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	/*
		if M.CheckSession(w, r) == false {
			http.Error(w, "unauthorised", http.StatusUnauthorized)
			return
		}
	*/
	vars := mux.Vars(r)
	processType := vars["ProcessType"]

	allowedContent := M.Config.AudioFormats
	reader, err := r.MultipartReader()
	if err != nil {
		fmt.Println("Error Reading multipart Reader ", err)
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

		var _user models.SubscriberUser
		if part.Header["Content-Type"][0] == "application/json" {
			jsonDecoder := json.NewDecoder(part)
			err = jsonDecoder.Decode(&_user)
			if err != nil {
				log.Println("Error in reading metadata ", err)
				continue
			}
			_, err = M.storage.SubscriberAdd(_user)
			if err != nil {
				log.Println("Error adding the user ", err)
			}
			continue
		}

		//if part.FileName() is empty, skip this iteration.
		if part.FileName() == "" {
			continue
		}

		filestatus.FileName = part.FileName()
		//if extension is not WAV skip for now.
		ext := filepath.Ext(filestatus.FileName)

		if ext != ".wav" {
			fmt.Printf("Processing file %s is no allowed yet !	 \n", ext)
			filestatus.Status = "Rejected"
			response.FileStatus = append(response.FileStatus, filestatus)
			continue
		}

		dst, err := os.Create(M.Config.UploadFolder + filestatus.FileName)

		defer dst.Close()
		if err != nil {
			log.Println("Error opening file")
			continue
		}

		buffer := make([]byte, 512)
		_cbytes, err := part.Read(buffer)

		//Here we send only 512 bytes to detect the right type
		content, err := GetFileContentType(buffer)
		if err != nil {
			log.Println("Error Getting Content ", err)
		}
		allowed := false
		for _, i := range allowedContent {
			if i == content {
				allowed = true
				break
			}
		}
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
		dst.Write(buffer[0:_cbytes])

		for {

			buffer := make([]byte, 100000)
			cBytes, err := part.Read(buffer)
			if err == io.EOF {
				log.Println("Saving Payload ID", payload.PayloadID)
				dst.Close()
				payload.CalculateHASH()
				filestatus.Hash = payload.FileHash
				filestatus.Status = "Uploaded"
				payload.ProcessPayload(processType)
				_user.FileNames = append(_user.FileNames, filestatus.FileName)
				//This call back suppose to update (key is the email)
				M.storage.SubscriberAdd(_user)
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

func getMetadata(r *http.Request) ([]byte, error) {
	f, _, err := r.FormFile("metadata")
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata form file: %v", err)
	}
	metadata, errRead := ioutil.ReadAll(f)
	if errRead != nil {
		return nil, fmt.Errorf("failed to read metadata: %v", errRead)
	}

	return metadata, nil
}

func setupResponse(w *http.ResponseWriter, req *http.Request) {
	log.Println("setting up")
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "admintoken, Content,Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}

func countryFlag(x string) string {
	if len(x) != 2 {
		return ""
	}
	if x[0] < 'A' || x[0] > 'Z' || x[1] < 'A' || x[1] > 'Z' {
		return ""
	}
	return string(0x1F1E6+rune(x[0])-'A') + string(0x1F1E6+rune(x[1])-'A')
}
