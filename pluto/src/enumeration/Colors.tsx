// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/enumeration/Colors.css";

import type { enumeration } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { color } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import { Color } from "@/color";
import { CSS } from "@/css";

export interface ColorsProps {
  entries: enumeration.Entry[];
}

/**
 * Shows the color of each entry as a dot. An entry with no color shows an empty dot.
 * Shows nothing when no entry has a color.
 */
export const Colors = ({ entries }: ColorsProps): ReactElement | null => {
  const colors = Color.useValues(entries.map((e) => e.color));
  if (entries.every((e) => e.color == null)) return null;
  return (
    <Flex.Box x align="center" gap={0.5} empty className={CSS.B("enumeration-colors")}>
      {entries.map((e, i) => {
        const c = colors[i];
        return (
          <span
            key={e.key}
            style={{ background: c == null ? undefined : color.cssString(c) }}
          />
        );
      })}
    </Flex.Box>
  );
};
