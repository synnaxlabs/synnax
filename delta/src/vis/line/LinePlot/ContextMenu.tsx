// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange } from "@synnaxlabs/x";

import { Divider, Menu as PMenu } from "@synnaxlabs/pluto";
import { Icon } from "@synnaxlabs/media";
import { Menu } from "@/components";

interface ContextMenuProps {
  scale?: Scale;
  selection: Box | null;
}

export const ContextMenu = ({ selection, scale }: ContextMenuProps): ReactElement => {
  const getTimeRange = (): TimeRange => {
    if (selection == null) throw new Error("Selection is null");
    if (scale == null) throw new Error("Scale is null");
    const scale_ = scale
      .translate(-selection.left)
      .magnify(1 / selection.width)
      .reverse();
    return new TimeRange(scale_.pos(0), scale_.pos(1));
  };

  return (
    <PMenu>
      {selection !== null && (
        <>
          <PMenu.Item
            size="small"
            itemKey="copyPython"
            startIcon={<Icon.Python />}
            onClick={() => {
              const tr = getTimeRange();
              const code = `synnax.TimeRange(${tr.start.valueOf()}, ${tr.end.valueOf()})`;
              void navigator.clipboard.writeText(code);
            }}
          >
            Copy Range as Python
          </PMenu.Item>
          <PMenu.Item
            size="small"
            itemKey="copyTS"
            startIcon={<Icon.Typescript />}
            onClick={() => {
              const tr = getTimeRange();
              const code = `new TimeRange(${tr.start.valueOf()}, ${tr.end.valueOf()})`;
              void navigator.clipboard.writeText(code);
            }}
          >
            Copy Range as TypeScript
          </PMenu.Item>
          <PMenu.Item
            size="small"
            itemKey="copyTS"
            startIcon={<Icon.Time />}
            onClick={() => {
              const tr = getTimeRange();
              const code = `${tr.start.fString("ISO", "local")} ${tr.end.fString(
                "ISO",
                "local"
              )}`;
              void navigator.clipboard.writeText(code);
            }}
          >
            Copy Range as ISO
          </PMenu.Item>
          <Divider direction="x" padded />
        </>
      )}
      <Menu.Item.HardReload />
    </PMenu>
  );
};
