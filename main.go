// Copyright 2017 Google Inc. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
//
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ericchiang/k8s"
	"github.com/ericchiang/k8s/api/v1"
	metav1 "github.com/ericchiang/k8s/apis/meta/v1"

	vaultapi "github.com/hashicorp/vault/api"
)

var (
	namespace  string
	secretName string
	vaultAddr  string
	vaultToken string
)

var (
	k8sClient   *k8s.Client
	vaultClient *vaultapi.Client

	secretPath string
)

func main() {
	flag.StringVar(&namespace, "namespace", "default", "The Kubernetes namespace.")
	flag.StringVar(&secretName, "secretName", "", "The Vault secret to sync.")
	flag.StringVar(&vaultAddr, "vault-addr", "http://vault:8200", "The address of the Vault server.")
	flag.StringVar(&vaultToken, "vault-token", "", "The Vault authentication token.")
	flag.Parse()

	var err error

	// Create a Kubernetes client.
	k8sClient, err = k8s.NewInClusterClient()
	if err != nil {
		log.Fatal(err)
	}

	k8sClient.Namespace = namespace

	// Create a Vault client.
	vaultClient, err = vaultapi.NewClient(vaultapi.DefaultConfig())
	if err != nil {
		log.Fatal(err)
	}

	err = vaultClient.SetAddress(vaultAddr)
	if err != nil {
		log.Fatal(err)
	}

	vaultClient.SetToken(vaultToken)

	secretPath = fmt.Sprintf("/secret/%s", secretName)

	log.Println("Syncing Vault secrets...")

	err = syncSecret()
	if err != nil {
		log.Println(err)
	}

	// Sync secrets in the background.
	go func() {
		for {
			time.Sleep(10 * time.Second)
			err := syncSecret()
			if err != nil {
				log.Println(err)
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Printf("Shutdown signal received, exiting...")
}

func syncSecret() error {
	vs, err := vaultClient.Logical().Read(secretPath)
	if err != nil {
		return err
	}

	if vs == nil {
		return fmt.Errorf("%s does not exist.", secretName)
	}

	stringData := make(map[string]string)
	for k, v := range vs.Data {
		stringData[k] = v.(string)
	}

	ks, err := k8sClient.CoreV1().GetSecret(context.TODO(), secretName, k8sClient.Namespace)
	if apiErr, ok := err.(*k8s.APIError); ok {
		if apiErr.Code == http.StatusNotFound {
			s := &v1.Secret{
				Metadata: &metav1.ObjectMeta{
					Name:      &secretName,
					Namespace: &k8sClient.Namespace,
				},
				StringData: stringData,
			}
			_, err := k8sClient.CoreV1().CreateSecret(context.TODO(), s)
			if err != nil {
				return err
			}
			return nil
		}
	}

	ks.StringData = stringData
	_, err = k8sClient.CoreV1().UpdateSecret(context.TODO(), ks)
	if err != nil {
		return err
	}

	return nil
}
