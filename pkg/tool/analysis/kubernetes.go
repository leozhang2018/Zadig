/*
Copyright 2023 The K8sGPT Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Some parts of this file have been modified to make it functional in Zadig

package analysis

import (
	"fmt"

	kube "github.com/koderover/zadig/v2/pkg/shared/kube/client"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/scheme"
)

type Client struct {
	Client        kubernetes.Interface
	RestClient    rest.Interface
	Config        *rest.Config
	ServerVersion *version.Info
}

func (c *Client) GetConfig() *rest.Config {
	return c.Config
}

func (c *Client) GetClient() kubernetes.Interface {
	return c.Client
}

func (c *Client) GetRestClient() rest.Interface {
	return c.RestClient
}

func NewClient(hubserverAddr, clusterID string) (*Client, error) {
	config, err := kube.GetRESTConfig(hubserverAddr, clusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rest config: %w", err)
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	config.APIPath = "/api"
	config.GroupVersion = &scheme.Scheme.PrioritizedVersionsForGroup("")[0]
	config.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: scheme.Codecs}

	restClient, err := rest.RESTClientFor(config)
	if err != nil {
		return nil, err
	}

	serverVersion, err := clientSet.ServerVersion()
	if err != nil {
		return nil, err
	}

	return &Client{
		Client:        clientSet,
		RestClient:    restClient,
		Config:        config,
		ServerVersion: serverVersion,
	}, nil
}
