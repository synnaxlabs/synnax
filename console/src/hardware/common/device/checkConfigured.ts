// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { z } from "zod";

export const checkConfigured = <
  Properties extends z.ZodType = z.ZodType,
  Make extends z.ZodType<string> = z.ZodString,
  Model extends z.ZodType<string> = z.ZodString,
>(device: device.Device<Properties, Make, Model>): void => {
  if (!device.configured) throw new Error(`${device.name} is not configured.`);
};
