// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/arc/graph/node/select/Select.css";

import { Minimal } from "@/arc/graph/node/Base";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Icon } from "@/icon";

export const Symbol = () => (
  <Minimal
    sinks={[{ key: "output", Icon: Icon.Boolean }]}
    centerSinks
    sources={[
      { key: "true", Icon: Icon.True },
      { key: "false", Icon: Icon.False },
    ]}
  >
    <Flex.Box className={CSS.BE("arc", "select-icon-frame")}>
      <Icon.Select className={CSS.BE("arc", "select-icon")} />
    </Flex.Box>
  </Minimal>
);
