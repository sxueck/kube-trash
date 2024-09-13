package config

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

type Cfg struct {
	APIServer   string
	KubeConfig  string
	MinioConfig MinioConfig
}

type MinioConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
}

var GlobalCfg = &Cfg{}

func InitConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.kube-record")
	viper.AddConfigPath("/config")
	viper.AutomaticEnv()

	// With the prefix set, Viper will look for an environment variable named "KR_APISERVER".
	viper.SetEnvPrefix("KR")
	viper.BindEnv("APIServer")
	// export KR_KUBECONFIG=~/.kube/config
	viper.BindEnv("KubeConfig")

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	err = viper.Unmarshal(GlobalCfg)
	if err != nil {
		log.Fatalf("Unable to decode into struct: %v", err)
	}

	fmt.Printf("%+v\n", GlobalCfg)
}

func init() {
	InitConfig()
	fmt.Printf("%+v\n", GlobalCfg)
}
