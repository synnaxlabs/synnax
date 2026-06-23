// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Config, defaultConfig } from "@/arc/graph/node/sink/config";
import { Form } from "@/arc/graph/node/sink/Form";
import { Symbol } from "@/arc/graph/node/sink/Sink";
import { type Spec } from "@/arc/graph/node/types/spec";

export { configZ } from "@/arc/graph/node/sink/config";

export const SPEC: Spec<"write", Config> = {
  key: "write",
  name: "Telemetry Sink",
  Form,
  Symbol,
  Preview: Symbol,
  defaultConfig,
  zIndex: 0,
};

export const REGISTRY = { write: SPEC };
