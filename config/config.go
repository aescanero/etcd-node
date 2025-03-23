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

	//Storage Configuration
	DataDir      string `env:"ETCD_DATA_DIR" default:"/var/run/etcd"`
	MaxSnapshots string `env:"ETCD_MAX_SNAPSHOTS" default:"5"`

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
	if os.Getenv("ETCD_CERT_FILE") == "" {
		addArg("cert-file", c.CertFile)
	}
	if os.Getenv("ETCD_KEY_FILE") == "" {
		addArg("key-file", c.KeyFile)
	}
	if os.Getenv("ETCD_TRUSTED_CA_FILE") == "" {
		addArg("trusted-ca-file", c.TrustedCAFile)
	}
	if os.Getenv("ETCD_PEER_CERT_FILE") == "" {
		addArg("peer-cert-file", c.PeerCertFile)
	}
	if os.Getenv("ETCD_PEER_KEY_FILE") == "" {
		addArg("peer-key-file", c.PeerKeyFile)
	}
	if os.Getenv("ETCD_PEER_TRUSTED_CA_FILE") == "" {
		addArg("peer-trusted-ca-file", c.PeerTrustedCAFile)
	}
	if os.Getenv("ETCD_CLIENT_CERT_AUTH") == "" {
		addArg("client-cert-auth", c.ClientCertAuth)
	}
	if os.Getenv("ETCD_PEER_CLIENT_CERT_AUTH") == "" {
		addArg("peer-client-cert-auth", c.PeerClientCertAuth)
	}

	// Node Configuration
	if os.Getenv("ETCD_NAME") == "" {
		addArg("name", c.Name)
	}
	if os.Getenv("ETCD_INITIAL_CLUSTER_STATE") == "" {
		addArg("initial-cluster-state", c.InitialClusterState)
	}
	if os.Getenv("ETCD_INITIAL_CLUSTER_TOKEN") == "" {
		addArg("initial-cluster-token", c.InitialClusterToken)
	}
	if os.Getenv("ETCD_INITIAL_CLUSTER") == "" {
		addArg("initial-cluster", c.InitialCluster)
	}
	if os.Getenv("ETCD_INITIAL_ADVERTISE_PEER_URLS") == "" {
		addArg("advertise-peer-urls", c.InitialAdvertisePeerURLs)
	}
	if os.Getenv("ETCD_LISTEN_PEER_URLS") == "" {
		addArg("listen-peer-urls", c.ListenPeerURLs)
	}
	if os.Getenv("ETCD_ADVERTISE_CLIENT_URLS") == "" {
		addArg("advertise-client-urls", c.AdvertiseClientURLs)
	}
	if os.Getenv("ETCD_LISTEN_CLIENT_URLS") == "" {
		addArg("listen-client-urls", c.ListenClientURLs)
	}
	if os.Getenv("ETCD_AUTO_COMPACTION_RETENTION") == "" {
		addArg("auto-compaction-retention", c.AutoCompactionRetention)
	}
	if os.Getenv("ETCD_QUOTA_BACKEND_BYTES") == "" {
		addArg("quota-backend-bytes", c.QuotaBackendBytes)
	}

	if os.Getenv("ETCD_DATA_DIR") == "" {
		addArg("data-dir", c.DataDir)
	}

	ListDataDir(c.DataDir)
	return args
}

// Function to print ETCD_DATA_DIR value and list files in that directory
func ListDataDir(dataDir string) error {
	// Print the data directory path
	fmt.Printf("ETCD_DATA_DIR: %s\n", dataDir)

	// Read directory contents
	files, err := os.ReadDir(dataDir)
	if err != nil {
		return fmt.Errorf("error reading directory %s: %v", dataDir, err)
	}

	// Print list of files
	fmt.Println("Files in directory:")
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			return fmt.Errorf("error getting file info for %s: %v", file.Name(), err)
		}

		// Print file name and permissions
		fmt.Printf("- %s (mode: %v)\n", file.Name(), info.Mode())
	}

	return nil
}
