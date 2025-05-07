// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { createContext, use } from "react";

export interface States extends Record<device.Key, device.State> {}

export const StateContext = createContext<States>({});

export const useState = (key: device.Key): device.State | undefined =>
  use(StateContext)[key];
