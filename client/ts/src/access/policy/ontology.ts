// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// This needs to be a separate file because these types are imported by the ontology
// module, but the policy module need to import the ID schema from the ontology module.

export const ONTOLOGY_TYPE = "policy";
export type OntologyType = typeof ONTOLOGY_TYPE;

export const ALLOW_ALL_ONTOLOGY_TYPE = "allow_all";
export type AllowAllOntologyType = typeof ALLOW_ALL_ONTOLOGY_TYPE;
