// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x/color";
import { Component } from "@synnaxlabs/charon/component";
import { text } from "@synnaxlabs/charon/text/base";

import { z } from "zod";

import { Control } from "@/schematic/node/common/control";
import { Label } from "@/schematic/node/common/label";
import { telem } from "@/telem/aether";
import { Button as ButtonTelem } from "@/vis/button";

export const VARIANT = "button" as const;

export const configZ = Label.labeledConfigZ.extend({
  variant: z.literal(VARIANT),
  size: Component.size.optional(),
  level: text.levelZ.optional(),
  onClickDelay: z.number().optional(),
  sink: telem.booleanSinkSpecZ.optional(),
  mode: z.enum(ButtonTelem.MODES).optional(),
  color: color.crudeZ.optional(),
  control: Control.stateConfigZ.optional(),
});
export type Config = z.infer<typeof configZ>;
