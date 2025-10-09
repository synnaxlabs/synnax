// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Constant } from "@/arc/stage/constant/Constant";
import { Form } from "@/arc/stage/constant/Form";
import { type types } from "@/arc/stage/types";

export const SYMBOLS: Record<string, types.Spec<any>> = {
  constant: {
    key: "constant",
    name: "Constant",
    zIndex: 100,
    Form,
    Symbol: Constant,
    Preview: Constant,
    defaultProps: () => ({ value: 0 }),
  },
};
