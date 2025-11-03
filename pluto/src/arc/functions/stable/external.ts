// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";

import { Form } from "@/arc/functions/stable/Form";
import { StableFor } from "@/arc/functions/stable/StableFor";
import { type types } from "@/arc/functions/types";

export const SYMBOLS: Record<string, types.Spec<any>> = {
  stable_for: {
    key: "stable_for",
    name: "Stable For",
    zIndex: 100,
    Form,
    Symbol: StableFor,
    Preview: StableFor,
    defaultProps: () => ({
      duration: TimeSpan.milliseconds(250).nanoseconds,
    }),
  },
};
