// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";

import { Minimal } from "@/arc/functions/Base";
import { Flex } from "@/flex";
import { Icon } from "@/icon";

const ORANGE_HEX = color.construct("#FF8A00");

export const Symbol = () => (
  <Minimal
    sinks={[{ key: "output", Icon: Icon.Boolean }]}
    centerSinks
    sources={[
      { key: "true", Icon: Icon.True },
      { key: "false", Icon: Icon.False },
    ]}
  >
    <Flex.Box
      style={{
        width: "5rem",
        overflow: "hidden",
      }}
    >
      <Icon.Select
        style={{
          width: "7rem",
          height: "7rem",
          color: color.cssString(ORANGE_HEX),
          strokeWidth: "1px",
          transform: "translateX(-20%)",
        }}
      />
    </Flex.Box>
  </Minimal>
);
