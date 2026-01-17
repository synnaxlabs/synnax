// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label } from "@synnaxlabs/x";

export type Params = label.Key | label.Key[];

export const keyZ = label.keyZ;
export type Key = label.Key;
export const labelZ = label.labelZ;
export type Label = label.Label;

export const LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE = "labeled_by";
