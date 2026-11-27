// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import type { Group } from "@/schematic/node/group";

export const GROUP: Group = {
  key: "pumps",
  scope: "schematic_symbol",
  name: "Pumps",
  Icon: Icon.Pump,
  symbols: [
    "cavity_pump",
    "centrifugal_compressor",
    "compressor",
    "diaphragm_pump",
    "ejection_pump",
    "ejector_compressor",
    "liquid_ring_compressor",
    "piston_pump",
    "pump",
    "roller_vane_compressor",
    "screw_pump",
    "turbo_compressor",
    "vacuum_pump",
  ],
};
