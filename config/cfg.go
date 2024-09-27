package config

import (
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
	KubeConfig string
	Storage    S3Config `mapstructure:"storage"`
	Watch      Watch    `mapstructure:"watch"`
}

type S3Config struct {
	Endpoint        string `mapstructure:"endpoint"`
	Port            int    `mapstructure:"port"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	Bucket          string `mapstructure:"bucket"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
}

type Watch struct {
	ExcludeResource []string `mapstructure:"exclude_resource"`
	IncludeResource []string `mapstructure:"include_resource"`
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
}
