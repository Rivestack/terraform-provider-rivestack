// Copyright (c) Rivestack
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func TestProviderSchema(t *testing.T) {
	// Verify the provider schema is valid by creating the server.
	resp := &tfprotov6.GetProviderSchemaResponse{}
	_ = resp

	_, err := providerserver.NewProtocol6WithError(New("test")())()
	if err != nil {
		t.Fatalf("unexpected error creating provider server: %s", err)
	}
}
