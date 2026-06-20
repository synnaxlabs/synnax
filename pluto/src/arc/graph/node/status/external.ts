// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Config, defaultConfig } from "@/arc/graph/node/status/config";
import { Form } from "@/arc/graph/node/status/Form";
import { Symbol } from "@/arc/graph/node/status/Change";
import { type Spec } from "@/arc/graph/node/types/spec";

export { configZ } from "@/arc/graph/node/status/config";

export const SPEC: Spec<"status.set", Config> = {
  key: "status.set",
  name: "Change Status",
  zIndex: 100,
  Form,
  Symbol,
  Preview: Symbol,
  defaultConfig,
};

export const REGISTRY = { "status.set": SPEC };
