// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Config, defaultConfig } from "@/arc/graph/node/stable/config";
import { Form } from "@/arc/graph/node/stable/Form";
import { StableFor } from "@/arc/graph/node/stable/StableFor";
import { type Spec } from "@/arc/graph/node/types/spec";

export { configZ } from "@/arc/graph/node/stable/config";

export const SPEC: Spec<"stable_for", Config> = {
  key: "stable_for",
  name: "Stable For",
  zIndex: 100,
  Form,
  Symbol: StableFor,
  Preview: StableFor,
  defaultConfig,
};

export const REGISTRY = { stable_for: SPEC };
