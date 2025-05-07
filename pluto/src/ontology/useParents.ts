// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";

import { useDependentTracker } from "@/ontology/useDependentTracker";

export const useParents = (
  id: ontology.CrudeID,
  filter?: (parent: ontology.Resource) => boolean,
): ontology.Resource[] =>
  useDependentTracker(id, ontology.FROM_RELATIONSHIP_DIRECTION, filter);
