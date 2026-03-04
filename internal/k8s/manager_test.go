// Copyright (c) 2026 Madalin Gabriel Ignisca <hi@madalin.me>
// Copyright (c) 2026 Vlah Software House SRL <contact@vlah.sh>
// All rights reserved. See LICENSE for details.

package k8s

import "testing"

func TestSanitizeDomain(t *testing.T) {
	tests := []struct {
		domain string
		want   string
	}{
		{"someblog.com", "custom-someblog-com"},
		{"my.cool.blog.io", "custom-my-cool-blog-io"},
		{"example.co.uk", "custom-example-co-uk"},
		{"simple", "custom-simple"},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			got := SanitizeDomain(tt.domain)
			if got != tt.want {
				t.Errorf("SanitizeDomain(%q) = %q, want %q", tt.domain, got, tt.want)
			}
		})
	}
}

func TestDisabledManagerIsNoop(t *testing.T) {
	m := NewManager("test-ns", false)
	if m.IsEnabled() {
		t.Error("expected disabled manager")
	}

	// All operations should succeed silently when disabled.
	if err := m.CreateDomainResources("example.com"); err != nil {
		t.Errorf("CreateDomainResources should not error when disabled: %v", err)
	}
	if err := m.DeleteDomainResources("example.com"); err != nil {
		t.Errorf("DeleteDomainResources should not error when disabled: %v", err)
	}
	if m.CertificateReady("example.com") {
		t.Error("CertificateReady should return false when disabled")
	}
}
