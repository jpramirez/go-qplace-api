package commands

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	models "github.com/jpramirez/go-qplace-api/pkg/models"

	"github.com/jpramirez/go-qplace-api/pkg/constants"
	utils "github.com/jpramirez/go-qplace-api/pkg/utils"
	"github.com/jpramirez/go-qplace-api/web"
)

var rootCmd = &cobra.Command{
	Use:   "WanPan",
	Short: "One Push notification Service",
	Long:  `A Fast and Flexible on push platform`,
	RunE:  runWebServer,
}

//Execute will run the desire module command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var config models.Config
var cfgFile string
var projectBase string
var monitorMode string

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config/config.json)")
	rootCmd.PersistentFlags().Bool("default", true, "Use default configuration")
}

func runWebServer(cmd *cobra.Command, args []string) error {
	config, err := utils.LoadConfiguration(cfgFile)
	if err != nil {
		log.Fatalln("Error and Exiting")
	}
	f, err := os.OpenFile(config.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	fmt.Println(config.LogFile)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	webagent, err := web.NewWebAgent(config, constants.BuildVersion, constants.BuildTime)
	if err != nil {
		log.Fatalln("Error on newebagent call ", err)
	}
	webagent.StartServer()
	return err
}
