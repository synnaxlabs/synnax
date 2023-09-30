// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type aether } from "@/aether/aether";
import { Controller } from "@/control/aether/controller";
import { Indicator } from "@/control/aether/indicator";
import { StateProvider } from "@/control/aether/state";

export const REGISTRY: aether.ComponentRegistry = {
  [Controller.TYPE]: Controller,
  [StateProvider.TYPE]: StateProvider,
  [Indicator.TYPE]: Indicator,
};
