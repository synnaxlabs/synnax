// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Key } from "@/label/types.gen";

/** One or many labels, by key. */
export type Params = Key | Key[];

/** Relationship type joining a resource to the labels on it. */
export const LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE = "labeled_by";
