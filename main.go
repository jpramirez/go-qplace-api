package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jpramirez/go-qplace-api/pkg/models"
)

var (
	MaxWorker = os.Getenv("MAX_WORKERS")
	MaxQueue  = os.Getenv("MAX_QUEUE")
)

// Job represents the job to be run
type Job struct {
	Payload models.Payload
}

// A buffered channel that we can send work requests on.
var JobQueue chan Job

// Worker represents the worker that executes the job
type Worker struct {
	WorkerPool chan chan Job
	JobChannel chan Job
	quit       chan bool
}

func NewWorker(workerPool chan chan Job) Worker {
	return Worker{
		WorkerPool: workerPool,
		JobChannel: make(chan Job),
		quit:       make(chan bool)}
}

// Start method starts the run loop for the worker, listening for a quit channel in
// case we need to stop it
func (w Worker) Start() {
	go func() {
		for {
			// register the current worker into the worker queue.
			w.WorkerPool <- w.JobChannel

			select {
			case job := <-w.JobChannel:
				// we have received a work request.
				if err := job.Payload.SavePayLoad; err != nil {
					log.Println("Error uploading to S3: %s", err)
				}

			case <-w.quit:
				// we have received a signal to stop
				return
			}
		}
	}()
}

type Dispatcher struct {
	// A pool of workers channels that are registered with the dispatcher
	WorkerPool chan chan Job
	maxWorkers int
	pool       int
}

func NewDispatcher(maxWorkers int) *Dispatcher {
	pool := make(chan chan Job, maxWorkers)
	return &Dispatcher{WorkerPool: pool}
}

func (d *Dispatcher) Run() {
	// starting n number of workers
	for i := 0; i < d.maxWorkers; i++ {
		worker := NewWorker(d.WorkerPool)
		worker.Start()
	}
	go d.dispatch()
}

func (d *Dispatcher) dispatch() {
	for {
		select {
		case job := <-JobQueue:
			// a job request has been received
			go func(job Job) {
				// try to obtain a worker job channel that is available.
				// this will block until a worker is idle
				jobChannel := <-d.WorkerPool

				// dispatch the job to the worker job channel
				jobChannel <- job
			}(job)
		}
	}
}

/*

func PayloadHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Read the body into a string for json decoding
	var content = &PayloadCollection{}
	err := json.NewDecoder(io.LimitReader(r.Body, MaxLength)).Decode(&content)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Go through each payload and queue items individually to be posted to S3
	for _, payload := range content.Payloads {

		// let's create a job with the payload
		work := Job{Payload: payload}

		// Push the work onto the queue.
		JobQueue <- work
	}

	w.WriteHeader(http.StatusOK)
}

*/

func renderError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(message))
}

//GetFileContentType will get the mime type of the file by reading its first 512 bytes (according to the standard)
func GetFileContentType(buffer []byte) (string, error) {

	// Use the net/http package's handy DectectContentType function. Always returns a valid
	// content-type by returning "application/octet-stream" if no others seemed to match.
	contentType := http.DetectContentType(buffer)

	return contentType, nil
}

//UploadHandler is in charge of optimizing upload request
func UploadHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	config := models.ServerConfig{
		MaxUploadSize: 2 * 1024 * 1024,
		UploadFolder:  "./upload/",
	}

	reader, err := r.MultipartReader()

	fmt.Println(err)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		//if part.FileName() is empty, skip this iteration.
		if part.FileName() == "" {
			continue
		}
		dst, err := os.Create(config.UploadFolder + part.FileName())
		defer dst.Close()

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		buffer := make([]byte, 512)

		part.Read(buffer)
		content, err := GetFileContentType(buffer)

		if content != "image/jpeg" {
			fmt.Println("format not supported ")
			continue
		}

		/*
			fmt.Println("Saving File")
			if _, err := io.Copy(dst, part); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		*/

		var read int64
		var p float32

		length := r.ContentLength

		ticker := time.Tick(time.Millisecond) // <-- use this in production
		//ticker := time.Tick(time.Second) // this is for demo purpose with longer delay

		for {

			buffer := make([]byte, 100000)
			cBytes, err := part.Read(buffer)
			if err == io.EOF {
				fmt.Printf("\nLast buffer read!")
				break
			}
			read = read + int64(cBytes)

			//fmt.Printf("\r read: %v  length : %v \n", read, length)

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

}

func main() {
	http.HandleFunc("/uploadAudio", UploadHandler)

	//static file handler.
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))

	//Listen on port 8080
	http.ListenAndServe(":8080", nil)
}
