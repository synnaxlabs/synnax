// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@/arc/stage/sink/Form";
import { type Config, Symbol } from "@/arc/stage/sink/Sink";
import { type Spec } from "@/arc/stage/types/spec";

export const SPEC: Spec<Config> = {
  key: "write",
  name: "Telemetry Sink",
  Form,
  Symbol,
  defaultProps: () => ({
    channel: 0,
    value: 0,
  }),
  Preview: Symbol,
  zIndex: 0,
};

export const SYMBOLS = {
  [SPEC.key]: SPEC,
};
