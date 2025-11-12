package config

import (
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"
)

type Server struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type Database struct {
	DbHost     string `yaml:"db_host"`
	DbPort     int    `yaml:"db_port"`
	DbName     string `yaml:"db_name"`
	DbUser     string `yaml:"db_user"`
	DbPassword string `yaml:"db_password"`
}

type RPC struct {
	RpcUrl      string `yaml:"rpc_url"`
	ChainId     uint64 `yaml:"chain_id"`
	NativeToken string `yaml:"native_token"`
	Decimal     uint8  `yaml:"decimal"`
}

type Symbols struct {
	Name    string `yaml:"name"`
	Decimal uint8  `yaml:"decimal"`
}

type Config struct {
	SkyeyeUrl      string        `yaml:"skyeye_url"`
	Server         Server        `yaml:"server"`
	Symbols        []Symbols     `yaml:"symbols"`
	RPCs           []*RPC        `yaml:"rpcs"`
	Metrics        Server        `yaml:"metrics"`
	MasterDb       Database      `yaml:"master_db"`
	SlaveDb        Database      `yaml:"slave_db"`
	SlaveDbEnable  bool          `yaml:"slave_db_enable"`
	EnableApiCache bool          `yaml:"enable_api_cache"`
	BackOffset     uint64        `yaml:"back_offset"`
	LoopInternal   time.Duration `yaml:"loop_internal"`
}

func New(path string) (*Config, error) {
	var config = new(Config)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}
	return config, nil
}
