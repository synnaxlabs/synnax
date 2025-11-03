// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, TimeSpan } from "@synnaxlabs/x";
import z from "zod/v4";

import { Base, type HandleSpec } from "@/arc/functions/Base";
import { type types } from "@/arc/functions/types";
import { Icon } from "@/icon";
import { Text } from "@/text";

export const configZ = z.object({
  duration: z.number(),
});

export type Config = z.infer<typeof configZ>;

export type SymbolProps = types.SymbolProps<Config>;

const PURPLE_HEX = color.construct("#635BFF");

const SINKS: HandleSpec[] = [{ key: "input", Icon: Icon.Value }];

const SOURCES: HandleSpec[] = [{ key: "output", Icon: Icon.Value }];

export const StableFor = ({ duration }: SymbolProps) => (
  <Base
    type="Stable For"
    Icon={<Icon.Time />}
    color={color.cssString(color.setAlpha(PURPLE_HEX, 0.2))}
    textColor={color.cssString(PURPLE_HEX)}
    sinks={SINKS}
    sources={SOURCES}
  >
    <Text.Text level="h4" weight={500} variant="code">
      {new TimeSpan(duration).toString()}
    </Text.Text>
  </Base>
);
