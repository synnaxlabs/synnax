// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Hardware } from "@/hardware";
import { Ontology } from "@/ontology";
import { Range } from "@/range";
import { Vis } from "@/vis";

export const DRAWER_ITEMS = [
  ...Hardware.NAV_DRAWER_ITEMS,
  Ontology.TOOLBAR,
  Range.TOOLBAR,
  Vis.TOOLBAR,
];
