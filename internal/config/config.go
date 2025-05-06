package config

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	KibanaURL    string `envconfig:"KIBANA_URL" default:"https://elasticsearch.service.ops.iptho.co.uk"`
	LDAPUsername string `envconfig:"LDAP_USERNAME"`
	LDAPPassword string `envconfig:"LDAP_PASSWORD"`
}

func Load() (*Config, error) {
	var c Config
	err := envconfig.Process("", &c)
	return &c, err
}
