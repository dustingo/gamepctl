package config

import (
	"io/ioutil"

	"github.com/pelletier/go-toml"
	"sigs.k8s.io/yaml"
)

// CmdConfig structure of yaml
type CmdConfig struct {
	Kind     string   `json:"kind"`
	Metadata Metadata `json:"metadata"`
	Spec     Spec     `json:"spec"`
	Check    Check    `json:"check"`
}

// Metadata metadata
type Metadata struct {
	User        string `json:"user"`
	Port        string `json:"port"`
	Timeout     int    `json:"timeout"`
	Concurrence int    `json:"concurrence"`
	Annotations string `json:"annotations"`
}

// Spec spec module
type Spec struct {
	PreExec []Settings `json:"preExec,omitempty"`
	Exec    []Settings `json:"exec,omitempty"`
}

// Settings settings of check module
type Settings struct {
	Server  string   `json:"server"` // hostname or ipaddress
	Type    string   `json:"type"`   // ssh,port,ps
	Command []string `json:"command,omitempty"`
	Wait    int      `json:"wait,omitempty"`
	Url     string   `json:"url,omitempty"`     //http url
	Process string   `json:"process,omitempty"` // process name
	Number  int      `json:"number,omitempty"`  // number of process
	Ports   []int    `json:"port,omitempty"`    // process ports
}

// Check check module
type Check struct {
	Data []Settings `json:"data"`
}

// LoadServerConfig 加载服务器资源文件
func LoadServerConfig(path string) (map[string]string, error) {
	servers := make(map[string]string)
	serverConfig, err := toml.LoadFile(path)
	if err != nil {
		return nil, err
	}
	var i interface{}
	err = serverConfig.Unmarshal(&i)
	if err != nil {
		return nil, err
	}
	for _, v := range i.(map[string]interface{}) {
		for name, ip := range v.(map[string]interface{}) {
			servers[name] = ip.(string)
		}
	}
	return servers, nil
}

func LoadCmdConfig(path string) (*CmdConfig, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg CmdConfig
	if err = yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
