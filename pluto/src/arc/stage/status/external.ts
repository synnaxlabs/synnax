// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Config, Symbol } from "@/arc/stage/status/Change";
import { Form } from "@/arc/stage/status/Form";
import { type Spec } from "@/arc/stage/types/spec";

export const SPEC: Spec<Config> = {
  key: "set_status",
  name: "Change Status ",
  zIndex: 100,
  Form,
  Symbol,
  defaultProps: () => ({
    statusKey: "",
    variant: "success",
    message: "Notification",
  }),
  Preview: Symbol,
};

export const SYMBOLS = {
  [SPEC.key]: SPEC,
};
