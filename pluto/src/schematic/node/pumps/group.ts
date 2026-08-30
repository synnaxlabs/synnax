// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@/icon";
import { type Group } from "@/schematic/node/group";

export const GROUP: Group = {
  key: "pumps",
  name: "Pumps",
  Icon: Icon.Pump,
  symbols: [
    "pump",
    "screw_pump",
    "piston_pump",
    "cavity_pump",
    "diaphragm_pump",
    "ejection_pump",
    "vacuum_pump",
    "compressor",
    "turbo_compressor",
    "roller_vane_compressor",
    "liquid_ring_compressor",
    "ejector_compressor",
    "centrifugal_compressor",
  ],
};
