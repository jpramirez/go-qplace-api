package models

type ServerConfig struct {
	MaxUploadSize int64
	UploadFolder  string
}

//Config Main Configuration File Structure
type Config struct {
	WebPort      string   `json:"webport"`
	WebAddress   string   `json:"webaddress"`
	DatabaseName string   `json:"databasename"`
	APIURL       string   `json:"apiurl"`
	AppName      string   `json:"appname"`
	LogFile      string   `json:"logfile"`
	KeyFile      string   `json:"keyfile"`
	UploadFolder string   `json:"uploadfolder"`
	CrtFile      string   `json:"crtfile"`
	AudioFormats []string `json:"audioformats"`
}
