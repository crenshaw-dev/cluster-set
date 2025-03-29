package argocd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crenshaw-dev/cluster-set/api/v1alpha1"
)

func ClusterTemplateToCluster(t v1alpha1.ClusterTemplate) (*Cluster, error) {
	c := Cluster{
		Server:  t.Spec.Server,
		Name:    t.Spec.Name,
		Project: t.Spec.Project,
	}

	config := ClusterConfig{
		ProxyUrl: t.Spec.ProxyUrl,
	}

	if t.Spec.AWSAuthConfig != nil {
		config.AWSAuthConfig = &AWSAuthConfig{
			ClusterName: t.Spec.AWSAuthConfig.ClusterName,
			RoleARN:     t.Spec.AWSAuthConfig.RoleARN,
			Profile:     t.Spec.AWSAuthConfig.Profile,
		}
	}

	tlsClientConfig := TLSClientConfig{
		ServerName: t.Spec.TLSClientConfig.ServerName,
		CertData:   []byte(t.Spec.TLSClientConfig.CertData),
		KeyData:    []byte(t.Spec.TLSClientConfig.KeyData),
		CAData:     []byte(t.Spec.TLSClientConfig.CAData),
	}
	if t.Spec.TLSClientConfig.Insecure != "" && t.Spec.TLSClientConfig.Insecure != "true" && t.Spec.TLSClientConfig.Insecure != "false" {
		return nil, fmt.Errorf("invalid value for TLSClientConfig.Insecure: %s", t.Spec.TLSClientConfig.Insecure)
	}
	config.TLSClientConfig.Insecure = t.Spec.TLSClientConfig.Insecure == "true"
	config.TLSClientConfig = tlsClientConfig

	if t.Spec.ExecProviderConfig != nil {
		execProviderConfig := ExecProviderConfig{
			Command:     t.Spec.ExecProviderConfig.Command,
			APIVersion:  t.Spec.ExecProviderConfig.APIVersion,
			InstallHint: t.Spec.ExecProviderConfig.InstallHint,
		}

		if t.Spec.ExecProviderConfig.Args != "" {
			var args []string
			err := json.Unmarshal([]byte(t.Spec.ExecProviderConfig.Args), &args)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal execProviderConfig args: %w", err)
			}
			execProviderConfig.Args = args
		}
		if t.Spec.ExecProviderConfig.Env != "" {
			var env map[string]string
			err := json.Unmarshal([]byte(t.Spec.ExecProviderConfig.Env), &env)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal execProviderConfig env: %w", err)
			}
			execProviderConfig.Env = env
		}

		config.ExecProviderConfig = &execProviderConfig
	}

	if t.Spec.DisableCompression != "" && t.Spec.DisableCompression != "true" && t.Spec.DisableCompression != "false" {
		return nil, fmt.Errorf("invalid value for disableCompression: %s", t.Spec.DisableCompression)
	}
	config.DisableCompression = t.Spec.DisableCompression == "true"

	if t.Spec.Namespaces != "" {
		var namespaces []string
		err := json.Unmarshal([]byte(t.Spec.Namespaces), &namespaces)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal namespaces: %w", err)
		}
		c.Namespaces = strings.Join(namespaces, ",")
	}

	if t.Spec.ClusterResources != "" && t.Spec.ClusterResources != "true" && t.Spec.ClusterResources != "false" {
		return nil, fmt.Errorf("invalid value for clusterResources: %s", t.Spec.ClusterResources)
	}
	c.ClusterResources = t.Spec.ClusterResources

	c.Project = t.Spec.Project

	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cluster config: %w", err)
	}
	c.Config = string(configJSON)

	return &c, nil
}

func ClusterTemplateToSecret(t v1alpha1.ClusterTemplate) (*corev1.Secret, error) {
	cluster, err := ClusterTemplateToCluster(t)
	if err != nil {
		return nil, fmt.Errorf("failed to convert cluster template to cluster: %w", err)
	}

	clusterJSON, err := json.Marshal(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cluster config: %w", err)
	}

	data := map[string]string{}
	err = json.Unmarshal(clusterJSON, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal cluster config: %w", err)
	}

	// base64-encode each data value
	secretData := map[string][]byte{}
	for key, value := range data {
		secretData[key] = []byte(base64.StdEncoding.EncodeToString([]byte(value)))
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        t.Metadata.Name,
			Labels:      t.Metadata.Labels,
			Annotations: t.Metadata.Annotations,
		},
		Data: secretData,
	}
	return secret, nil
}

// Cluster is the definition of a cluster resource
type Cluster struct {
	// Server is the API server URL of the Kubernetes cluster
	Server string `json:"server"`
	// Name of the cluster. If omitted, will use the server address
	Name string `json:"name"`
	// Config holds cluster information for connecting to a cluster
	Config string `json:"config"`
	// Holds list of namespaces which are accessible in that cluster. Cluster level resources will be ignored if namespace list is not empty.
	Namespaces string `json:"namespaces,omitempty"`
	// Indicates if cluster level resources should be managed. This setting is used only if cluster is connected in a namespaced mode.
	ClusterResources string `json:"clusterResources,omitempty"`
	// Reference between project and cluster that allow you automatically to be added as item inside Destinations project entity
	Project string `json:"project,omitempty"`
}

// AWSAuthConfig is an AWS IAM authentication configuration
type AWSAuthConfig struct {
	// ClusterName contains AWS cluster name
	ClusterName string `json:"clusterName,omitempty"`

	// RoleARN contains optional role ARN. If set then AWS IAM Authenticator assume a role to perform cluster operations instead of the default AWS credential provider chain.
	RoleARN string `json:"roleARN,omitempty"`

	// Profile contains optional role ARN. If set then AWS IAM Authenticator uses the profile to perform cluster operations instead of the default AWS credential provider chain.
	Profile string `json:"profile,omitempty"`
}

// ExecProviderConfig is config used to call an external command to perform cluster authentication
// See: https://godoc.org/k8s.io/client-go/tools/clientcmd/api#ExecConfig
type ExecProviderConfig struct {
	// Command to execute
	Command string `json:"command,omitempty"`

	// Arguments to pass to the command when executing it
	Args []string `json:"args,omitempty"`

	// Env defines additional environment variables to expose to the process
	Env map[string]string `json:"env,omitempty"`

	// Preferred input version of the ExecInfo
	APIVersion string `json:"apiVersion,omitempty"`

	// This text is shown to the user when the executable doesn't seem to be present
	InstallHint string `json:"installHint,omitempty"`
}

// ClusterConfig is the configuration attributes. This structure is subset of the go-client
// rest.Config with annotations added for marshalling.
type ClusterConfig struct {
	// Server requires Basic authentication
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`

	// Server requires Bearer authentication. This client will not attempt to use
	// refresh tokens for an OAuth2 flow.
	// TODO: demonstrate an OAuth2 compatible client.
	BearerToken string `json:"bearerToken,omitempty"`

	// TLSClientConfig contains settings to enable transport layer security
	TLSClientConfig `json:"tlsClientConfig"`

	// AWSAuthConfig contains IAM authentication configuration
	AWSAuthConfig *AWSAuthConfig `json:"awsAuthConfig,omitempty"`

	// ExecProviderConfig contains configuration for an exec provider
	ExecProviderConfig *ExecProviderConfig `json:"execProviderConfig,omitempty"`

	// DisableCompression bypasses automatic GZip compression requests to the server.
	DisableCompression bool `json:"disableCompression,omitempty"`

	// ProxyURL is the URL to the proxy to be used for all requests send to the server
	ProxyUrl string `json:"proxyUrl,omitempty"` //nolint:revive //FIXME(var-naming)
}

// TLSClientConfig contains settings to enable transport layer security
type TLSClientConfig struct {
	// Insecure specifies that the server should be accessed without verifying the TLS certificate. For testing only.
	Insecure bool `json:"insecure"`
	// ServerName is passed to the server for SNI and is used in the client to check server
	// certificates against. If ServerName is empty, the hostname used to contact the
	// server is used.
	ServerName string `json:"serverName,omitempty"`
	// CertData holds PEM-encoded bytes (typically read from a client certificate file).
	// CertData takes precedence over CertFile
	CertData []byte `json:"certData,omitempty"`
	// KeyData holds PEM-encoded bytes (typically read from a client certificate key file).
	// KeyData takes precedence over KeyFile
	KeyData []byte `json:"keyData,omitempty"`
	// CAData holds PEM-encoded bytes (typically read from a root certificates bundle).
	// CAData takes precedence over CAFile
	CAData []byte `json:"caData,omitempty"`
}
