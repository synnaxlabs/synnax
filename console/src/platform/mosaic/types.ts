// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { type location } from "@synnaxlabs/x";

export interface OnDropParams {
  id: ontology.ID;
  nodeKey: number;
  location: location.Location;
}

export interface DropHandler {
  (params: OnDropParams): void;
}
