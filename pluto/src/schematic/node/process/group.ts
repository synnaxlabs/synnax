// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/charon";
import { type Group } from "@/schematic/node/group";

export const GROUP: Group = {
  key: "process",
  name: "Process",
  Icon: Icon.Process,
  symbols: [
    "heatExchangerGeneral",
    "heatExchangerM",
    "heatExchangerStraightTube",
    "staticMixer",
    "rotaryMixer",
    "agitator",
    "propellerAgitator",
    "flatBladeAgitator",
    "paddleAgitator",
    "crossBeamAgitator",
    "helicalAgitator",
  ],
};
