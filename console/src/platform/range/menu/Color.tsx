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
import { useSyncedRef } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Theming } from "@synnaxlabs/lyra/theming";
import { Color as PColor, Ranger } from "@synnaxlabs/pluto";
import { color } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useState } from "react";

interface PickerProps {
  value?: color.Crude;
  fallback: color.Crude;
  onCommit: (value?: color.Color) => void;
}

// The picker changes on each tick of a drag, so the pick commits once, when the
// submenu or the menu closes and unmounts the picker.
const Picker = ({ value, fallback, onCommit }: PickerProps): ReactElement => {
  const [pending, setPending] = useState<{ value?: color.Color } | null>(null);
  const pendingRef = useSyncedRef(pending);
  const onCommitRef = useSyncedRef(onCommit);
  useEffect(
    () => () => {
      if (pendingRef.current != null) onCommitRef.current(pendingRef.current.value);
    },
    [pendingRef, onCommitRef],
  );
  return (
    <Base.Picker
      value={pending == null ? value : pending.value}
      fallback={fallback}
      onChange={(next) => setPending({ value: next })}
    />
  );
};

export interface ColorProps {
  ranges: ranger.Range[];
}

/** Sets the color of the ranges with the picker that the range swatch opens. */
export const Color = ({ ranges }: ColorProps): ReactElement => {
  const { update } = Ranger.useSetColor();
  const { colors } = Theming.use();
  const resolve = PColor.useResolve();
  const values = ranges.map((r) => resolve(r.color));
  const first = values[0];
  const shared = values.every((v) =>
    v == null || first == null ? v === first : color.equals(v, first),
  );
  return (
    <Menu.Sub itemKey="color" textValue="Color">
      <Menu.SubTrigger>
        <Icon.Circle color={shared ? (first ?? colors.gray.l8) : colors.gray.l8} />
        Color
      </Menu.SubTrigger>
      <Menu.Dialog itemless aria-label="Color">
        <Picker
          value={shared ? first : undefined}
          fallback={colors.gray.l8}
          onCommit={(next) =>
            ranges.forEach((r) =>
              update({
                key: r.key,
                color: next == null ? undefined : { type: "fixed", value: next },
              }),
            )
          }
        />
      </Menu.Dialog>
    </Menu.Sub>
  );
};
