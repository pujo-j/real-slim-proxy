package cmd

import (
	"fmt"
	"github.com/pujo-j/real-slim-proxy/proxy"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

var cfgFile string
var debug bool
var cloudType string
var config proxy.ServerConfig

var rootCmd = &cobra.Command{
	Use:   "real-slim-proxy",
	Short: "real-slim-proxy is a dependency caching proxy with object-store artifact storage",
	Long: `real-slim-proxy is a dependency caching proxy with object-store artifact storage
It's designed for use in ephemeral cloud builders as a sidecar or pseudo-sidecar and avoid repeated internet downloads`,
	Run: func(cmd *cobra.Command, args []string) {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		server, err := proxy.NewServer(config)
		if err != nil {
			log.WithError(err).Panic("building server")
		}
		go func() {
			s := <-sigs
			log.WithField("signal", s).Debug("Shutting down")
			_ = server.Server.Close()
		}()
		log.Infof("Starting server on %v", server.Server.Addr)
		err = server.Server.ListenAndServe()
		if err != nil {
			if err == http.ErrServerClosed {
				log.Info("Server shutdown complete")
			} else {
				log.WithError(err).Error("starting server")
			}
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $PWD/slim.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&debug, "verbose", "v", false, "Verbose Logging")
}

func initConfig() {
	if debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	if cfgFile == "" {
		cwd, err := os.Getwd()
		if err != nil {
			// WTF, no CWD ?
			log.Panic(err)
		}
		cfgFile = filepath.Join(cwd, "slim.yaml")
	}

	log.WithField("configFile", cfgFile).Debug("Reading config")
	configFile, err := os.Open(cfgFile)
	if err != nil {
		log.Panic(err)
	}
	configBytes, err := ioutil.ReadAll(configFile)
	if err != nil {
		log.Panic(err)
	}
	config = proxy.ServerConfig{}
	err = yaml.Unmarshal(configBytes, &config)
	if err != nil {
		log.Panic(err)
	}
	log.WithField("configFile", cfgFile).WithField("config", config).Debug("Config read success")
}
