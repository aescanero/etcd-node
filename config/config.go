/*Copyright [2023] [Alejandro Escanero Blanco <aescanero@disasterproject.com>]

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.*/

/*

ports:
  ldap: 1389
  ldaps: 1686
  ca.pem: ...
  crt.pem: ...
  crt.key: ...
  ca.pem.file: ...
  crt.pem.file: ...
  crt.key.file: ...
databases:
- base: ...
  replicas:
  - url:
    ca.pem: ...
    crt.pem: ...
    crt.key: ...
	ca.pem.file: ...
    crt.pem.file: ...
    crt.key.file: ...
	attrs: "*,+"


*/
/*
olcSynrepl:
syncrepl
 ...
 provider=ldaps://ldap.example.com
 bindmethod=simple
 binddn="cn=goodguy,dc=example,dc=com"
 credentials=dirtysecret
 starttls=critical
 schemachecking=on
 scope=sub
 searchbase="dc=example,dc=com"
 tls_cacert=/path/to/file
 tls_cert=/path/to/file.ext
 tls_key=/path/to/file.ext
 tls_protocol_min=1.2
 tls_reqcert=demand
 type=refreshAndPersist

*/
package config

import (
	"fmt"
	"os"
	"reflect"
	"strings"
)

type Config struct {
	// TLS Configuration
	CertFile           string `env:"ETCD_CERT_FILE"`
	KeyFile            string `env:"ETCD_KEY_FILE"`
	TrustedCAFile      string `env:"ETCD_TRUSTED_CA_FILE"`
	PeerCertFile       string `env:"ETCD_PEER_CERT_FILE"`
	PeerKeyFile        string `env:"ETCD_PEER_KEY_FILE"`
	PeerTrustedCAFile  string `env:"ETCD_PEER_TRUSTED_CA_FILE"`
	ClientCertAuth     bool   `env:"ETCD_CLIENT_CERT_AUTH"`
	PeerClientCertAuth bool   `env:"ETCD_PEER_CLIENT_CERT_AUTH"`

	// Node Configuration
	Name                     string `env:"ETCD_NAME"`
	InitialClusterState      string `env:"ETCD_INITIAL_CLUSTER_STATE"`
	InitialClusterToken      string `env:"ETCD_INITIAL_CLUSTER_TOKEN"`
	InitialCluster           string `env:"ETCD_INITIAL_CLUSTER"`
	InitialAdvertisePeerURLs string `env:"ETCD_INITIAL_ADVERTISE_PEER_URLS"`
	ListenPeerURLs           string `env:"ETCD_LISTEN_PEER_URLS"`
	AdvertiseClientURLs      string `env:"ETCD_ADVERTISE_CLIENT_URLS"`
	ListenClientURLs         string `env:"ETCD_LISTEN_CLIENT_URLS"`

	// Performance Configuration
	AutoCompactionRetention string `env:"ETCD_AUTO_COMPACTION_RETENTION"`
	QuotaBackendBytes       string `env:"ETCD_QUOTA_BACKEND_BYTES"`

	RootUser     string `env:"ETCD_ROOT_USER"`
	RootPassword string `env:"ETCD_ROOT_PASSWORD"`

	ReadUser     string `env:"ETCD_ROOT_USER"`
	ReadPassword string `env:"ETCD_ROOT_PASSWORD"`
}

func NewConfig() *Config {
	return new(Config)
}

func LoadFromEnv() (*Config, error) {
	config := &Config{}

	// Utilizamos reflection para cargar las variables de entorno
	v := reflect.ValueOf(config).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		envTag := field.Tag.Get("env")
		if envTag == "" {
			continue
		}

		envValue := os.Getenv(envTag)
		fieldValue := v.Field(i)

		switch field.Type.Kind() {
		case reflect.String:
			fieldValue.SetString(envValue)
		case reflect.Bool:
			fieldValue.SetBool(strings.ToLower(envValue) == "true")
		}
	}

	return config, nil
}

func (c *Config) BuildArgs() []string {
	args := []string{}

	// Función helper para agregar argumentos
	addArg := func(name string, value interface{}) {
		if value != nil && value != "" {
			switch v := value.(type) {
			case bool:
				if v {
					args = append(args, fmt.Sprintf("--%s=%t", name, v))
				}
			default:
				args = append(args, fmt.Sprintf("--%s=%v", name, v))
			}
		}
	}

	// TLS Configuration
	addArg("cert-file", c.CertFile)
	addArg("key-file", c.KeyFile)
	addArg("trusted-ca-file", c.TrustedCAFile)
	addArg("peer-cert-file", c.PeerCertFile)
	addArg("peer-key-file", c.PeerKeyFile)
	addArg("peer-trusted-ca-file", c.PeerTrustedCAFile)
	addArg("client-cert-auth", c.ClientCertAuth)
	addArg("peer-client-cert-auth", c.PeerClientCertAuth)

	// Node Configuration
	addArg("name", c.Name)
	addArg("initial-cluster-state", c.InitialClusterState)
	addArg("initial-cluster-token", c.InitialClusterToken)
	addArg("initial-cluster", c.InitialCluster)
	addArg("initial-advertise-peer-urls", c.InitialAdvertisePeerURLs)
	addArg("listen-peer-urls", c.ListenPeerURLs)
	addArg("advertise-client-urls", c.AdvertiseClientURLs)
	addArg("listen-client-urls", c.ListenClientURLs)

	// Performance Configuration
	addArg("auto-compaction-retention", c.AutoCompactionRetention)
	addArg("quota-backend-bytes", c.QuotaBackendBytes)

	return args
}
