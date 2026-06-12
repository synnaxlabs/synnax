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
  key: "general",
  name: "General",
  Icon: Icon.Channel,
  symbols: [
    "value",
    "gauge",
    "setpoint",
    "text_box",
    "off_page_reference",
    "button",
    "select",
    "switch",
    "input",
    "light",
    "state_indicator",
    "polygon",
    "circle",
    "box",
  ],
};
