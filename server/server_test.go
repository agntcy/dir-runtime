// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package server

import "testing"

func TestValidateSchemaURL(t *testing.T) {
	tests := []struct {
		url     string
		wantErr bool
	}{
		{"https://schema.oasf.outshift.com", false},
		{"https://schema.example.com/oasf", false},
		{"http://localhost:8080", false},
		{"http://127.0.0.1:8080", false},
		{"http://[::1]:8080", false},
		{"http://schema.example.com", true},
		{"http://10.0.0.5:8080", true},
		{"ftp://schema.example.com", true},
		{"schema.oasf.outshift.com", true},
		{"https://user@schema.example.com", true},
		{"https://schema.example.com?x=1", true},
		{"", true},
	}

	for _, tt := range tests {
		if err := ValidateSchemaURL(tt.url); (err != nil) != tt.wantErr {
			t.Errorf("ValidateSchemaURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
		}
	}
}

func TestWithAcceptedSchemaURLs_RejectsInvalidURL(t *testing.T) {
	o := &options{}

	if err := WithAcceptedSchemaURLs([]string{"http://schema.example.com"})(o); err == nil {
		t.Fatal("want an error for plain http on a non-loopback host")
	}

	if err := WithAcceptedSchemaURLs([]string{"http://localhost:8080"})(o); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(o.acceptedSchemaURLs) != 1 {
		t.Fatalf("accepted URLs not set: %v", o.acceptedSchemaURLs)
	}
}
