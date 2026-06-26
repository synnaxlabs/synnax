// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Config, defaultConfig } from "@/arc/graph/node/read/config";
import { Form } from "@/arc/graph/node/read/Form";
import { Symbol } from "@/arc/graph/node/read/Read";
import { type Spec } from "@/arc/graph/node/types/spec";

export { configZ } from "@/arc/graph/node/read/config";

export const SPEC: Spec<"telem.read", Config> = {
  key: "telem.read",
  name: "Read Channel",
  Form,
  Symbol,
  Preview: Symbol,
  defaultConfig,
  zIndex: 0,
};

export const REGISTRY = { "telem.read": SPEC };
