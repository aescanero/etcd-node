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

package cmd

import (
	"github.com/aescanero/etcd-node/utils"
	"github.com/spf13/cobra"
)

var (
	port string
)

func init() {
	statusCmd.Flags().StringVarP(&port, "port", "p", utils.GetEnv("LDAP_PORT", "1389"), "LDAP port (default 1389)")
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Etcd Node Status",
	Long:  `Etcd Node Status`,
	Run: func(cmd *cobra.Command, args []string) {
		app.IsRunning()
	},
}
