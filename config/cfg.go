package config

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/viper"
)

// QueueItem represents an item in the queue
type QueueItem struct {
	Data      []byte
	Kind      string
	Name      string
	Namespace string
}

type Cfg struct {
	APIServer  string      `mapstructure:"api_server"`
	KubeConfig string      `mapstructure:"kube_config"`
	Storage    MinioConfig `mapstructure:"storage"`
}

type MinioConfig struct {
	Endpoint        string `mapstructure:"endpoint"`
	Port            int    `mapstructure:"port"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	Bucket          string `mapstructure:"bucket"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
}

var GlobalCfg = &Cfg{}

func InitConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.kube-trash")
	viper.AddConfigPath("/config")
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	err = viper.Unmarshal(GlobalCfg)
	if err != nil {
		log.Fatalf("Unable to decode into struct: %v", err)
	}

	GlobalCfg.KubeConfig = os.Getenv("KR_KUBECONFIG")
	fmt.Printf("%+v\n", GlobalCfg)
}

func init() {
	InitConfig()
	fmt.Printf("%+v\n", GlobalCfg)
}
