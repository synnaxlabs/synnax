// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/task/fields/StateUpdateRate.css";

import { CSS } from "@/platform/css";
import { Rate } from "@/platform/task/fields/Rate";

export const StateUpdateRate = () => (
  <Rate
    label="State Update Rate"
    path="config.stateRate"
    className={CSS.BM("task-rate-field", "fixed")}
  />
);
