// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type z } from "zod";

import { type Layout } from "@/hardware/common/task/Form";
import { type Import } from "@/import";

export const createIngestor =
  (configSchema: z.ZodType, zeroLayout: Layout): Import.FileIngestor =>
  (data: unknown, { layout, placeLayout }) => {
    const config = configSchema.parse(data);
    placeLayout({ ...zeroLayout, ...layout, key: layout.key, args: { config } });
  };
