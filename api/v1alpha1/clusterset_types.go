/*
Copyright 2025.

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

package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterSetSpec defines the desired state of ClusterSet.
type ClusterSetSpec struct {
	// Generator is a generator that produces a list of parameters for templating clusters.
	Generator ClusterSetGenerator `json:"generator,omitempty"`

	// Template is a template of a cluster to be generated.
	// +kubebuilder:validation:Required
	Template ClusterTemplate `json:"template"`
}

// ClusterSetStatus defines the observed state of ClusterSet.
type ClusterSetStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ClusterSet is the Schema for the clustersets API.
type ClusterSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSetSpec   `json:"spec,omitempty"`
	Status ClusterSetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterSetList contains a list of ClusterSet.
type ClusterSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterSet{}, &ClusterSetList{})
}

type ClusterSetGenerator struct {
	// List is a list generator, used to define static lists of clusters.
	List *ListGenerator `json:"list,omitempty"`
}

type ListGenerator struct {
	// Elements is a static list of elements to be produced by the list generator.
	Elements []apiextensionsv1.JSON `json:"elements,omitempty"`
}

type ClusterTemplate struct {
	// Metadata is the metadata of the cluster template.
	// +kubebuilder:validation:Required
	Metadata ClusterTemplateMetadata `json:"metadata"`

	// Spec is the specification of the cluster template.
	// +kubebuilder:validation:Required
	Spec ClusterTemplateSpec `json:"spec"`
}

type ClusterTemplateMetadata struct {
	// Name is the name of the cluster Secret.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Labels for cluster secret metadata. Must be a JSON or YAML string.
	Labels string `json:"labels,omitempty"`

	// Annotations for cluster secret metadata. Must be a JSON or YAML string.
	Annotations string `json:"annotations,omitempty"`
}

// nolint:revive
/**
ClusterTemplateSpec is basically a copy of the Cluster struct from Argo CD, reformatted a bit to make it easier to
template over. For example, instead of being a JSON string, the `config` field is an actual struct. And instead of being
arrays, maps, and booleans, fields of those types are strings so that they can be templated.

Using strings for these fields makes it easier to template over them (it avoids the `templatePatch` method used in
ApplicationSets). But it also makes it more difficult to convert the template field to a Cluster CRD in the future if
we want to do that. The good news is that the conversions from arrays, maps, and booleans strings to their actual types
is fairly easy. There's just the chance that something is malformatted, and the conversion fails.
*/

type ClusterTemplateSpec struct {
	// Server is the URL of the Kubernetes API server.
	// +kubebuilder:validation:Required
	Server string `json:"server"`
	// Name is the name of the cluster.
	Name string `json:"name,omitempty"`

	// Username is the username for basic authentication.
	Username string `json:"username,omitempty"`
	// Password is the password for basic authentication.
	Password string `json:"password,omitempty"`

	// BearerToken is the bearer token for authentication.
	BearerToken string `json:"bearerToken,omitempty"`

	// TLSClientConfig is the TLS client configuration for the cluster.
	TLSClientConfig TLSClientConfig `json:"tlsClientConfig,omitempty"`

	// AWSAuthConfig contains IAM authentication configuration
	AWSAuthConfig *AWSAuthConfig `json:"awsAuthConfig,omitempty"`

	// ExecProviderConfig contains configuration for an exec provider.
	ExecProviderConfig *ExecProviderConfig `json:"execProviderConfig,omitempty"`

	// DisableCompression bypasses automatic GZip compression requests to the server. Must be exactly either "true" or
	// "false".
	// +kubebuilder:validation:Enum=true;false
	DisableCompression string `json:"disableCompression,omitempty"`

	// ProxyUrl is the URL to the proxy to be used for all requests send to the server.
	ProxyUrl string `json:"proxyUrl,omitempty"`

	// Namespaces holds list of namespaces which are accessible in that cluster. Cluster level resources will be ignored
	// if namespace list is not empty. Must be formatted as a YAML- or JSON-encoded array of strings.
	Namespaces string `json:"namespaces,omitempty"`

	// ClusterResources indicates if cluster level resources should be managed. This setting is used only if cluster is
	// connected in a namespaced mode. Must be either "true" or "false".
	// +kubebuilder:validation:Enum=true;false
	ClusterResources string `json:"clusterResources,omitempty"`

	// Project is a reference between project and cluster that allow you automatically to be added as item inside
	// Destinations project entity.
	Project string `json:"project,omitempty"`
}

// TLSClientConfig contains settings to enable transport layer security
type TLSClientConfig struct {
	// Insecure specifies that the server should be accessed without verifying the TLS certificate. For testing only.
	// Must be exactly either "true" or "false".
	// +kubebuilder:validation:Enum=true;false
	Insecure string `json:"insecure,omitempty"`

	// ServerName is passed to the server for SNI and is used in the client to check server
	// certificates against. If ServerName is empty, the hostname used to contact the
	// server is used.
	ServerName string `json:"serverName,omitempty"`

	// CertData holds a PEM-encoded string (typically read from a client certificate file).
	CertData string `json:"certData,omitempty"`

	// KeyData holds a PEM-encoded bytes (typically read from a client certificate key file).
	KeyData string `json:"keyData,omitempty"`

	// CAData holds a PEM-encoded bytes (typically read from a root certificates bundle).
	CAData string `json:"caData,omitempty"`
}

// AWSAuthConfig is an AWS IAM authentication configuration
type AWSAuthConfig struct {
	// ClusterName contains AWS cluster name
	ClusterName string `json:"clusterName,omitempty"`

	// RoleARN contains optional role ARN. If set then AWS IAM Authenticator assume a role to perform cluster operations
	// instead of the default AWS credential provider chain.
	RoleARN string `json:"roleARN,omitempty"`

	// Profile contains optional role ARN. If set then AWS IAM Authenticator uses the profile to perform cluster
	// operations instead of the default AWS credential provider chain.
	Profile string `json:"profile,omitempty"`
}

// ExecProviderConfig is config used to call an external command to perform cluster authentication
// See: https://godoc.org/k8s.io/client-go/tools/clientcmd/api#ExecConfig
type ExecProviderConfig struct {
	// Command to execute
	Command string `json:"command,omitempty"`

	// Args is a list of arguments to pass to the command when executing it. Must be formatted as a YAML- or JSON-
	// encoded array of strings.
	Args string `json:"args,omitempty"`

	// Env defines additional environment variables to expose to the process. Must be formatted as a YAML- or JSON-
	// encoded map of strings to strings.
	Env string `json:"env,omitempty"`

	// Preferred input version of the ExecInfo
	APIVersion string `json:"apiVersion,omitempty"`

	// This text is shown to the user when the executable doesn't seem to be present
	InstallHint string `json:"installHint,omitempty"`
}
