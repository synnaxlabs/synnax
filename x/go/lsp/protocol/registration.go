// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package protocol

// Registration general parameters to register for a capability.
type Registration struct {
	// ID is the id used to register the request. The id can be used to deregister
	// the request again.
	ID string `json:"id"`

	// Method is the method / capability to register for.
	Method string `json:"method"`

	// RegisterOptions options necessary for the registration.
	RegisterOptions interface{} `json:"registerOptions,omitempty"`
}

// RegistrationParams params of Register Capability.
type RegistrationParams struct {
	Registrations []Registration `json:"registrations"`
}

// TextDocumentRegistrationOptions TextDocumentRegistration options.
type TextDocumentRegistrationOptions struct {
	// DocumentSelector a document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector DocumentSelector `json:"documentSelector"`
}

// Unregistration general parameters to unregister a capability.
type Unregistration struct {
	// ID is the id used to unregister the request or notification. Usually an id
	// provided during the register request.
	ID string `json:"id"`

	// Method is the method / capability to unregister for.
	Method string `json:"method"`
}

// UnregistrationParams params of Unregistration.
type UnregistrationParams struct {
	Unregisterations []Unregistration `json:"unregisterations"`
}
