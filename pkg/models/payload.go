package models

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"time"

	ulid "github.com/epyphite/ulid"

	speech "cloud.google.com/go/speech/apiv1"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

//Payload model
type Payload struct {
	PayloadID     string
	StorageFolder string
	Content       []byte
	PayloadName   string
	PayloadType   string
	FileHash      string
}

var ulidSource *ulid.MonotonicULIDsource

func (P *Payload) Recongnize() error {
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

func (P *Payload) SavePayLoad() error {
	var err error
	return err
}

func (P *Payload) CalculateHASH() {
	fmt.Println(P.StorageFolder + "/" + P.PayloadName)
	P.FileHash = hex.EncodeToString(ComputeMD5(P.StorageFolder + "/" + P.PayloadName))
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
