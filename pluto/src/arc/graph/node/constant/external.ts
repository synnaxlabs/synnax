// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Config, defaultConfig } from "@/arc/graph/node/constant/config";
import { Constant } from "@/arc/graph/node/constant/Constant";
import { Form } from "@/arc/graph/node/constant/Form";
import { type Spec } from "@/arc/graph/node/types/spec";

export { configZ } from "@/arc/graph/node/constant/config";

export const SPEC: Spec<"constant", Config> = {
  key: "constant",
  name: "Constant",
  zIndex: 100,
  Form,
  Symbol: Constant,
  Preview: Constant,
  defaultConfig,
};

export const REGISTRY = { constant: SPEC };
