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
import { Controller } from "@/telem/control/aether/controller";
import { Indicator } from "@/telem/control/aether/indicator";
import { Legend } from "@/telem/control/aether/legend";
import { StateProvider } from "@/telem/control/aether/state";

export const REGISTRY: aether.ComponentRegistry = {
  [Controller.TYPE]: Controller,
  [StateProvider.TYPE]: StateProvider,
  [Indicator.TYPE]: Indicator,
  [Chip.TYPE]: Chip,
  [Legend.TYPE]: Legend,
};
