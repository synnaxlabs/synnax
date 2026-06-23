// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, TimeSpan } from "@synnaxlabs/x";

import { Base, type HandleSpec } from "@/arc/graph/node/Base";
import { type Config } from "@/arc/graph/node/stable/config";
import { type types } from "@/arc/graph/node/types";
import { Icon } from "@/icon";
import { Text } from "@/text";

export type SymbolProps = types.SymbolProps<Config>;

const PURPLE_HEX = color.construct("#635BFF");

const SINKS: HandleSpec[] = [{ key: "input", Icon: Icon.Value }];

const SOURCES: HandleSpec[] = [{ key: "output", Icon: Icon.Value }];

export const StableFor = ({ config: { duration } }: SymbolProps) => (
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
