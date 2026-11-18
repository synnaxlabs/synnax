// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { Color as Base } from "@synnaxlabs/lyra/color";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Theming } from "@synnaxlabs/lyra/theming";
import { Ranger } from "@synnaxlabs/pluto";
import { color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

export interface ColorProps {
  ranges: ranger.Range[];
}

/** Sets the color of the ranges with the picker that the range swatch opens. */
export const Color = ({ ranges }: ColorProps): ReactElement => {
  const { update } = Ranger.useSetColor();
  const { colors } = Theming.use();
  const first = ranges[0]?.color;
  const shared = ranges.every((r) =>
    r.color == null || first == null ? r.color === first : color.equals(r.color, first),
  );
  return (
    <Menu.Sub itemKey="color" textValue="Color">
      <Menu.SubTrigger>
        <Icon.Circle color={shared ? (first ?? colors.gray.l8) : colors.gray.l8} />
        Color
      </Menu.SubTrigger>
      <Menu.Dialog itemless aria-label="Color">
        <Base.Picker
          value={shared ? first : undefined}
          fallback={colors.gray.l8}
          onChange={(next) =>
            ranges.forEach((r) => update({ key: r.key, color: next }))
          }
        />
      </Menu.Dialog>
    </Menu.Sub>
  );
};
