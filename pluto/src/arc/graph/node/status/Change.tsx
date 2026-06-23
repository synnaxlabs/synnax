// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";

import { Base } from "@/arc/graph/node/Base";
import { type Config } from "@/arc/graph/node/status/config";
import { type types } from "@/arc/graph/node/types";
import { Icon } from "@/icon";
import { Status } from "@/status";
import { Text } from "@/text";

const RED_HEX = color.construct("#DC136C");

export interface SymbolProps extends types.SymbolProps<Config> {}

export const Symbol = ({ config: { message, variant } }: SymbolProps) => (
  <Base
    type="Change Status"
    Icon={<Icon.Notification />}
    color={color.cssString(color.setAlpha(RED_HEX, 0.2))}
    textColor={color.cssString(RED_HEX)}
    sinks={[{ key: "output", Icon: Icon.Value }]}
  >
    <Text.Text level="p" weight={500} color={11}>
      <Status.Indicator variant={variant} size="2.5em" />
      {message}
    </Text.Text>
  </Base>
);
