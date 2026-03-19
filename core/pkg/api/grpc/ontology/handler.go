// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ontology

import (
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/v1"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
)

func IDToProto(id ontology.ID) *gapi.OntologyID {
	if id.IsZero() {
		return nil
	}
	return &gapi.OntologyID{Type: string(id.Type), Key: id.Key}
}

func IDFromProto(id *gapi.OntologyID) ontology.ID {
	if id == nil {
		return ontology.ID{}
	}
	return ontology.ID{Type: ontology.Type(id.Type), Key: id.Key}
}
