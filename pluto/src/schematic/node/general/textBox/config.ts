// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { z } from "zod";

import { Flex } from "@/flex";
import { Label } from "@/schematic/node/common/label";
import { text } from "@/text/base";

export const VARIANT = "textBox" as const;

export const configZ = Label.labeledConfigZ.extend({
  variant: z.literal(VARIANT),
  color: color.crudeZ.optional(),
  width: z.number().optional(),
  align: Flex.alignmentZ.optional(),
  autoFit: z.boolean().optional(),
  level: text.levelZ.optional(),
  value: z.string().optional(),
});
export type Config = z.infer<typeof configZ>;
