// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type aether } from "@/aether/aether";
import { Chip } from "@/telem/control/aether/chip";
import { Colors } from "@/telem/control/aether/colors";
import { Controller } from "@/telem/control/aether/controller";
import { Indicator } from "@/telem/control/aether/indicator";
import { Legend } from "@/telem/control/aether/legend";

export const REGISTRY: aether.ComponentRegistry = {
  [Controller.TYPE]: Controller,
  [Colors.TYPE]: Colors,
  [Indicator.TYPE]: Indicator,
  [Chip.TYPE]: Chip,
  [Legend.TYPE]: Legend,
};
