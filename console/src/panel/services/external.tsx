// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { Ontology } from "@/ontology";

// Minimal panel ontology service so the console's per-resource service registry
// compiles. The full panel ontology service (icon resolution, drag/drop ingest,
// context menu, etc.) lands when the panel UI ships.
export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "panel",
  icon: <Icon.Panel />,
};
