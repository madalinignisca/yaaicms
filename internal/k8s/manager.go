// Copyright (c) 2026 Madalin Gabriel Ignisca <hi@madalin.me>
// Copyright (c) 2026 Vlah Software House SRL <contact@vlah.sh>
// All rights reserved. See LICENSE for details.

// Package k8s manages Kubernetes resources for custom domain TLS provisioning.
// It creates cert-manager Certificate CRs and Traefik IngressRoute CRs using
// the raw K8s REST API with in-cluster ServiceAccount authentication.
package k8s

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

const (
	// In-cluster ServiceAccount paths.
	saTokenPath  = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	saCACertPath = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"

	// Default K8s API server address (in-cluster).
	defaultAPIServer = "https://kubernetes.default.svc"

	// HTTP-01 ClusterIssuer used for custom domain certificates.
	httpIssuerName = "letsencrypt-prod"
)

// Manager creates and deletes K8s resources (Certificate + IngressRoute) for
// custom domains. It uses the in-cluster ServiceAccount token for auth.
type Manager struct {
	apiServer string
	token     string
	caCert    []byte
	namespace string
	service   string
	port      int
	enabled   bool
	client    *http.Client
}

// NewManager creates a K8s resource manager. If enabled is false, all
// operations are no-ops (for local development).
func NewManager(namespace string, enabled bool) *Manager {
	m := &Manager{
		apiServer: defaultAPIServer,
		namespace: namespace,
		service:   "yaaicms",
		port:      80,
		enabled:   enabled,
	}

	if !enabled {
		slog.Info("k8s manager disabled (dev mode)")
		return m
	}

	// Read ServiceAccount credentials.
	token, err := os.ReadFile(saTokenPath)
	if err != nil {
		slog.Error("failed to read SA token — k8s operations will fail", "error", err)
		return m
	}
	m.token = string(token)

	caCert, err := os.ReadFile(saCACertPath)
	if err != nil {
		slog.Error("failed to read SA CA cert — k8s operations will fail", "error", err)
		return m
	}
	m.caCert = caCert

	// Build TLS client with the cluster CA.
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caCert)
	m.client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: pool,
			},
		},
	}

	slog.Info("k8s manager initialized", "namespace", namespace)
	return m
}

// IsEnabled returns whether K8s resource management is active.
func (m *Manager) IsEnabled() bool {
	return m.enabled
}

// SanitizeDomain converts a domain name to a K8s-safe resource name.
// "someblog.com" → "custom-someblog-com"
func SanitizeDomain(domain string) string {
	return "custom-" + strings.ReplaceAll(domain, ".", "-")
}

// CreateDomainResources creates a cert-manager Certificate CR and a Traefik
// IngressRoute CR for the given custom domain.
func (m *Manager) CreateDomainResources(domain string) error {
	if !m.enabled {
		slog.Info("k8s: skipping resource creation (disabled)", "domain", domain)
		return nil
	}

	name := SanitizeDomain(domain)

	// Create Certificate CR.
	if err := m.createCertificate(name, domain); err != nil {
		return fmt.Errorf("create certificate for %s: %w", domain, err)
	}

	// Create IngressRoute CR.
	if err := m.createIngressRoute(name, domain); err != nil {
		return fmt.Errorf("create ingressroute for %s: %w", domain, err)
	}

	slog.Info("k8s resources created", "domain", domain, "name", name)
	return nil
}

// DeleteDomainResources removes the Certificate and IngressRoute for a domain.
func (m *Manager) DeleteDomainResources(domain string) error {
	if !m.enabled {
		slog.Info("k8s: skipping resource deletion (disabled)", "domain", domain)
		return nil
	}

	name := SanitizeDomain(domain)

	// Delete IngressRoute (ignore not-found).
	_ = m.deleteResource(
		fmt.Sprintf("/apis/traefik.io/v1alpha1/namespaces/%s/ingressroutes/%s", m.namespace, name),
	)

	// Delete Certificate (ignore not-found — cert-manager will clean up the secret).
	_ = m.deleteResource(
		fmt.Sprintf("/apis/cert-manager.io/v1/namespaces/%s/certificates/%s", m.namespace, name),
	)

	// Also try deleting the TLS secret directly.
	_ = m.deleteResource(
		fmt.Sprintf("/api/v1/namespaces/%s/secrets/%s-tls", m.namespace, name),
	)

	slog.Info("k8s resources deleted", "domain", domain, "name", name)
	return nil
}

// CertificateReady checks if the TLS secret for a domain exists, which
// indicates cert-manager has successfully issued the certificate.
func (m *Manager) CertificateReady(domain string) bool {
	if !m.enabled {
		return false
	}

	name := SanitizeDomain(domain)
	path := fmt.Sprintf("/api/v1/namespaces/%s/secrets/%s-tls", m.namespace, name)

	resp, err := m.doRequest("GET", path, nil)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// createCertificate creates a cert-manager Certificate CR for HTTP-01 validation.
func (m *Manager) createCertificate(name, domain string) error {
	cert := map[string]any{
		"apiVersion": "cert-manager.io/v1",
		"kind":       "Certificate",
		"metadata": map[string]any{
			"name":      name,
			"namespace": m.namespace,
		},
		"spec": map[string]any{
			"secretName": name + "-tls",
			"issuerRef": map[string]any{
				"name": httpIssuerName,
				"kind": "ClusterIssuer",
			},
			"dnsNames": []string{domain},
		},
	}

	body, _ := json.Marshal(cert)
	path := fmt.Sprintf("/apis/cert-manager.io/v1/namespaces/%s/certificates", m.namespace)

	resp, err := m.doRequest("POST", path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		// Already exists — not an error.
		return nil
	}
	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("k8s API %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// createIngressRoute creates a Traefik IngressRoute CR for the custom domain.
func (m *Manager) createIngressRoute(name, domain string) error {
	route := map[string]any{
		"apiVersion": "traefik.io/v1alpha1",
		"kind":       "IngressRoute",
		"metadata": map[string]any{
			"name":      name,
			"namespace": m.namespace,
		},
		"spec": map[string]any{
			"entryPoints": []string{"websecure"},
			"routes": []map[string]any{
				{
					"match": fmt.Sprintf("Host(`%s`)", domain),
					"kind":  "Rule",
					"services": []map[string]any{
						{
							"name": m.service,
							"port": m.port,
						},
					},
				},
			},
			"tls": map[string]any{
				"secretName": name + "-tls",
			},
		},
	}

	body, _ := json.Marshal(route)
	path := fmt.Sprintf("/apis/traefik.io/v1alpha1/namespaces/%s/ingressroutes", m.namespace)

	resp, err := m.doRequest("POST", path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		return nil
	}
	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("k8s API %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// deleteResource sends a DELETE request to the given K8s API path.
func (m *Manager) deleteResource(path string) error {
	resp, err := m.doRequest("DELETE", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("k8s API %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// doRequest performs an authenticated HTTP request to the K8s API server.
func (m *Manager) doRequest(method, path string, body []byte) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, m.apiServer+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("build k8s request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+m.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return m.client.Do(req)
}
