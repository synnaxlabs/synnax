// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/version/BetaTag.css";

import { Tag, Text, Tooltip } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/css";

export const BetaTag = (): ReactElement => (
  <Tooltip.Dialog>
    <Text.Text level="small" shade={8} style={{ width: 100, whiteSpace: "wrap" }}>
      This feature is still in development and may not always work as expected.
    </Text.Text>
    <Tag.Tag variant="filled" className={CSS.B("beta-tag")} icon={undefined}>
      Beta
    </Tag.Tag>
  </Tooltip.Dialog>
);
