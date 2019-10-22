package models

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	ulid "github.com/epyphite/ulid"

	speech "cloud.google.com/go/speech/apiv1"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

//Payload Actions
type PayloadAction struct {
	ActionName    string
	ActionStatus  string
	ActionMessage string
}

//Payload model
type Payload struct {
	PayloadID     string
	StorageFolder string
	PayloadName   string
	PayloadType   string
	FileHash      string
	Actions       []PayloadAction
}

var ulidSource *ulid.MonotonicULIDsource

//Recognize will send the file for recognition to google api.
func (P *Payload) Recognize() error {
	ctx := context.Background()

	client, err := speech.NewClient(ctx)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Recognizing File , opening")

	audioData, err := ioutil.ReadFile(P.StorageFolder + "/" + P.PayloadName)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Open file")

	response, err := client.Recognize(ctx, &speechpb.RecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:        speechpb.RecognitionConfig_LINEAR16,
			SampleRateHertz: 16000,
			LanguageCode:    "en-US",
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{Content: audioData},
		},
	})

	fmt.Println("Function Called")
	if err != nil {
		fmt.Println(err)
	}

	for _, result := range response.Results {
		fmt.Println("Results")
		for _, alt := range result.Alternatives {
			fmt.Println(alt.Words)
			fmt.Println(alt.Transcript)
		}
	}

	fmt.Println("Exiting")

	return err
}

//ProcessPayload will process and send the needed
func (P *Payload) ProcessPayload(process string) error {
	var err error
	fmt.Println("Process Payload Processing")

	url := "http://localhost:3000/api/v1/process/" + process + "/" + P.FileHash
	ret := P.sendPostRequest(url, P.StorageFolder+"/"+P.PayloadName, "files")
	var _action PayloadAction

	_action.ActionName = process

	if process == "splitfile" {

		dst, err := os.Create(P.StorageFolder + P.FileHash + ".zip") //We save the file from the return function
		_, err = dst.Write(ret)

		if err != nil {
			return err
		}
		dst.Close()
		//Return the zip file of split files.
		//We save it.
		_action.ActionStatus = "Completed"
		_action.ActionMessage = P.StorageFolder + P.FileHash + ".zip"
	} else if process == "recognize" {
		_action.ActionStatus = "Completed"
		_action.ActionMessage = string(ret)
		fmt.Println(string(ret))
	} else if process == "convert" {
		dst, err := os.Create(P.StorageFolder + P.FileHash + ".wav") //We save the file from the return function
		_, err = dst.Write(ret)
		if err != nil {
			return err
		}
		dst.Close()
		_action.ActionStatus = "Completed"
		_action.ActionMessage = P.StorageFolder + P.FileHash + ".wav"
	}

	P.Actions = append(P.Actions, _action)

	return err
}

//CalculateHASH will calculate the hash of the file.
func (P *Payload) CalculateHASH() {
	fmt.Println(P.StorageFolder + "/" + P.PayloadName)
	P.FileHash = hex.EncodeToString(ComputeMD5(P.StorageFolder + "/" + P.PayloadName))
}

//sendPostRequest Internal function for send files to the rest of the api's
func (P *Payload) sendPostRequest(url string, filename string, filetype string) []byte {
	file, err := os.Open(filename)

	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(filetype, filepath.Base(file.Name()))

	if err != nil {
		fmt.Println(err)
	}
	io.Copy(part, file)
	writer.Close()
	request, err := http.NewRequest("POST", url, body)

	if err != nil {
		fmt.Println(err)
	}

	request.Header.Add("Content-Type", writer.FormDataContentType())
	client := &http.Client{}

	response, err := client.Do(request)

	if err != nil {
		fmt.Print(err)
	}
	defer response.Body.Close()

	content, err := ioutil.ReadAll(response.Body)

	if err != nil {
		fmt.Println(err)
	}

	return content
}

//SavePayLoad save placeHolder
func (P *Payload) SavePayLoad() error {
	var err error
	return err
}

//NewPayLoad  will create a new instance.
func NewPayLoad(name string, Type string) (Payload, error) {
	var payload Payload
	var err error
	entropy := rand.New(rand.NewSource(time.Unix(1000000, 0).UnixNano()))
	// reproducible entropy source

	// sub-ms safe ULID generator
	ulidSource = ulid.NewMonotonicULIDsource(entropy)
	now := time.Now()
	ulidity, _ := ulidSource.New(now)

	payload.PayloadName = name
	payload.PayloadID = ulidity.String()
	payload.PayloadType = Type
	return payload, err
}

func ComputeMD5(filePath string) []byte {
	var result []byte
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Can't open the file")
		return nil
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil
	}
	return hash.Sum(result)
}
