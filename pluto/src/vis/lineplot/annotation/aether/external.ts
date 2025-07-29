// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type aether } from "@/aether/aether";
import { Provider } from "@/vis/lineplot/annotation/aether/provider";

export * from "@/vis/lineplot/annotation/aether/provider";

export const REGISTRY: aether.ComponentRegistry = {
  [Provider.TYPE]: Provider,
};
