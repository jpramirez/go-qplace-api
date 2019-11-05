package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/epyphite/ulid"
	c "github.com/jpramirez/go-qplace-api/pkg/crypto"
	models "github.com/jpramirez/go-qplace-api/pkg/models"
	uuid "github.com/satori/go.uuid"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/crypto/bcrypt"
)

//Client holds the Struct for boltDB
type Client struct {
	Database string
	boltDB   *bolt.DB
}

var ulidSource *ulid.MonotonicULIDsource

//OpenBoltDb main structure to open
func (bc *Client) OpenBoltDb(dataDir string, dataDbName string) *Client {

	Client := new(Client)
	var err error

	log.Printf("Opening Database %s, %s \n", dataDir, dataDbName)
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		os.Mkdir(dataDir, 0770)
	}

	var databaseFileName = ""

	databaseFileName = dataDir + string(os.PathSeparator) + dataDbName

	Client.Database = databaseFileName
	Client.boltDB, err = bolt.Open(databaseFileName, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	// reproducible entropy source
	entropy := rand.New(rand.NewSource(time.Unix(1000000, 0).UnixNano()))

	// sub-ms safe ULID generator
	ulidSource = ulid.NewMonotonicULIDsource(entropy)

	return Client
}

//Seed is good for creating basic buckets
func (bc *Client) Seed() {
	bc.initializeBucket()
}

//CloseDB will close the FD to the boltdb file
func (bc *Client) CloseDB() {
	bc.boltDB.Close()
}

//initializeBucket will setup file and buckets in qplaceDB.
//This is a boltdb key/Value Storage
// All collections are created here if they dont exists
func (bc *Client) initializeBucket() {

	err := bc.boltDB.Update(func(tx *bolt.Tx) error {
		root, err := tx.CreateBucketIfNotExists([]byte("qplace"))
		if err != nil {
			return fmt.Errorf("could not create root bucket: %v", err)
		}
		_, err = root.CreateBucketIfNotExists([]byte("users"))
		if err != nil {
			return fmt.Errorf("could not create weight bucket: %v", err)
		}
		_, err = root.CreateBucketIfNotExists([]byte("Payloads"))
		if err != nil {
			return fmt.Errorf("could not create Agents bucket: %v", err)
		}
		_, err = root.CreateBucketIfNotExists([]byte("Registry"))
		if err != nil {
			return fmt.Errorf("could not create Registry bucket: %v", err)
		}
		_, err = root.CreateBucketIfNotExists([]byte("Tokens"))
		if err != nil {
			return fmt.Errorf("could not create Tokens bucket: %v", err)
		}
		_, err = root.CreateBucketIfNotExists([]byte("Subscribers"))
		if err != nil {
			return fmt.Errorf("could not create Tokens bucket: %v", err)
		}
		return nil

	})
	if err != nil {
		fmt.Printf("%s", err.Error())
	}

	var adminToken models.Token
	u2, err := uuid.NewV4()
	if err != nil {
		log.Println("UserAdd -> Something went wrong: ", err)
		return
	}
	now := time.Now()
	adminToken.TokenID = u2.String()
	adminToken.IsAdmin = true
	adminToken.TimeStamp = now.UnixNano()
	bc.TokenAdd(adminToken)

	fmt.Println("A new Administation token has been added, use it to create System users")
	fmt.Println(adminToken.TokenID)
	/*
		var tempUser models.User
		// or error handling
		u2, err := uuid.NewV4()
		if err != nil {
			log.Println("UserAdd -> Something went wrong: ", err)
			return
		}
		tempUser.Username = "root"
		tempUser.Email = "jramirez@epyphite.com"
		tempUser.Password = []byte("qplace2020!!") //Default Password CHANGE IN PROD
		tempUser.UserID = u2.String()
		tempUser.Token = ""
		tempUser.Approved = false
		tempUser.Banned = true
		tempUser.Role = "Admin"
		err = bc.UserAdd(tempUser)
		if err != nil {
			fmt.Printf("%s", err.Error())
		}
	*/
}

//CheckToken for checkig validity of admin token
func (bc *Client) CheckToken(tokenstring string) (models.Token, error) {
	var _token models.Token
	err := bc.boltDB.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys

		b := tx.Bucket([]byte("qplace")).Bucket([]byte("users"))

		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var token models.Token
			json.Unmarshal(v, &token)
			if token.TokenID == tokenstring {

				_token = token
			}

		}
		return nil
	})
	return _token, err
}

//TokenAdd will add a token to the tokens bucket
func (bc *Client) TokenAdd(token models.Token) error {
	//log.Println("UserAdd --> password ", user.Password)
	tokenBytes, err := json.Marshal(&token)

	if err != nil {
		return fmt.Errorf("could not marshal config proto: %v", err)
	}
	err = bc.boltDB.Update(func(tx *bolt.Tx) error {
		err = tx.Bucket([]byte("qplace")).Bucket([]byte("users")).Put([]byte(token.TokenID), tokenBytes)
		if err != nil {
			fmt.Printf("%s", err.Error())
			return fmt.Errorf("could not set Token: %v", err)
		}
		return nil
	})
	return nil
}

//UserAdd will add a user using models.users
func (bc *Client) UserAdd(user models.User) error {
	_, err := bc.CheckUserCExists(&user)
	if err != nil {
		return fmt.Errorf("could not marshal config proto: %v", err)
	}
	// or error handling
	u2, err := uuid.NewV4()
	if err != nil {
		log.Println("UserAdd -> Something went wrong: ", err)
		return err
	}

	user.UserID = u2.String()
	user.Token = c.CreateHash(user.Email)

	user.Password, _ = bcrypt.GenerateFromPassword(user.Password, bcrypt.DefaultCost)
	//log.Println("UserAdd --> password ", user.Password)
	userBytes, err := json.Marshal(&user)

	if err != nil {
		return fmt.Errorf("could not marshal config proto: %v", err)
	}
	err = bc.boltDB.Update(func(tx *bolt.Tx) error {
		err = tx.Bucket([]byte("qplace")).Bucket([]byte("users")).Put([]byte(user.Email), userBytes)
		if err != nil {
			fmt.Printf("%s", err.Error())
			return fmt.Errorf("could not set USER: %v", err)
		}
		return nil
	})
	return nil
}

func decodeUser(data []byte) (models.User, error) {
	var p models.User
	err := json.Unmarshal(data, &p)
	if err != nil {
		return p, err
	}
	return p, nil
}

//CheckUserCExists usage is to check the hash of the IOC and search in the database.
func (bc *Client) CheckUserCExists(user *models.User) (*models.User, error) {

	var iuser models.User
	err := bc.boltDB.View(func(tx *bolt.Tx) error {
		iochhash := tx.Bucket([]byte("qplace")).Bucket([]byte("users")).Get([]byte(user.Email))
		if len(iochhash) > 0 {
			var err error
			iuser, err = decodeUser(iochhash)
			if err != nil {
				return fmt.Errorf("Bucket exists 1")
			}
			return nil
		}
		return nil
	})

	return &iuser, err
}

//CheckUser usage for authentication, returns true or false.
func (bc *Client) CheckUser(user models.User) (models.User, bool, error) {

	var iuser models.User
	err := bc.boltDB.View(func(tx *bolt.Tx) error {
		iochhash := tx.Bucket([]byte("qplace")).Bucket([]byte("users")).Get([]byte(user.Email))
		if len(iochhash) > 0 {
			var err error
			iuser, err = decodeUser(iochhash)
			if err != nil {
				return fmt.Errorf("Bucket exists 1")
			}
			return nil
		}
		return fmt.Errorf("Bucket exists 2	")
	})
	if err != nil {
		return iuser, false, err
	}

	err = bcrypt.CompareHashAndPassword(iuser.Password, user.Password)

	if err == nil {
		return iuser, true, err
	}
	//log.Println("Error in compare password ", err)
	return iuser, false, err
}

//CheckUserByID
func (bc *Client) CheckUserByID(userID string) (models.User, error) {
	var _user models.User
	err := bc.boltDB.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys

		b := tx.Bucket([]byte("qplace")).Bucket([]byte("users"))

		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var user models.User
			json.Unmarshal(v, &user)
			if user.UserID == userID {

				_user = user
			}

		}
		return nil
	})
	return _user, err
}

//PayloadAdd Adds a new discovered agent.
func (bc *Client) PayloadAdd(payload models.Payload) (models.Payload, error) {

	payloadBytes, err := json.Marshal(payload)

	err = bc.boltDB.Update(func(tx *bolt.Tx) error {
		err = tx.Bucket([]byte("qplace")).Bucket([]byte("Payloads")).Put([]byte(payload.PayloadID), payloadBytes)
		if err != nil {
			log.Printf("%s", err.Error())
			return fmt.Errorf("could not set Agent: %v", err)
		}
		return nil
	})
	return payload, nil
}

func (bc *Client) PayloadGetByID(payloadID string) (models.Payload, error) {
	var ipayload models.Payload
	err := bc.boltDB.View(func(tx *bolt.Tx) error {
		var err error
		bPayload := tx.Bucket([]byte("qplace")).Bucket([]byte("Payloads")).Get([]byte(payloadID))
		if len(bPayload) > 0 {
			err = json.Unmarshal(bPayload, &ipayload)
			if err != nil {
				return fmt.Errorf("Bucket exists 1")
			}
			return nil
		}
		return err
	})
	return ipayload, err

}

//SubscriberAdd Adds a new discovered agent.
func (bc *Client) SubscriberAdd(payload models.SubscriberUser) (models.SubscriberUser, error) {
	subscriberUserBytes, err := json.Marshal(payload)
	err = bc.boltDB.Update(func(tx *bolt.Tx) error {
		err = tx.Bucket([]byte("qplace")).Bucket([]byte("Subscribers")).Put([]byte(payload.Email), subscriberUserBytes)
		if err != nil {
			return fmt.Errorf("could not set Subscriber: %v", err)
		}
		return nil
	})
	return payload, err
}
