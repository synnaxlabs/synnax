// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Icon } from "@synnaxlabs/media";
import { Menu as Core } from "@synnaxlabs/pluto";
import { TimeRange, box, scale } from "@synnaxlabs/x";

import { Menu } from "@/components";
import { useSelectAxisBounds, useSelectSelection } from "@/lineplot/selectors";

export interface ContextMenuContentProps {
  layoutKey: string;
}

export const ContextMenuContent = ({
  layoutKey,
}: ContextMenuContentProps): ReactElement => {
  const { box: selection } = useSelectSelection(layoutKey);
  const bounds = useSelectAxisBounds(layoutKey, "x1");

  const s = scale.Scale.scale(1).scale(bounds);
  const timeRange = new TimeRange(
    s.pos(box.left(selection)),
    s.pos(box.right(selection)),
  );

  const handleSelect = (key: string): void => {
    switch (key) {
      case "iso":
        void navigator.clipboard.writeText(
          `${timeRange.start.fString("ISO")} - ${timeRange.end.fString("ISO")}`,
        );
        break;
      case "python":
        void navigator.clipboard.writeText(
          `sy.TimeRange(${timeRange.start.valueOf()}, ${timeRange.end.valueOf()})`,
        );
        break;
      case "typescript":
        void navigator.clipboard.writeText(
          `new TimeRange(${timeRange.start.valueOf()}, ${timeRange.end.valueOf()})`,
        );
        break;
    }
  };

  return (
    <Core.Menu onChange={handleSelect} iconSpacing="small">
      <Menu.Item.HardReload />
      {!box.areaIsZero(selection) && (
        <>
          <Core.Item itemKey="iso" startIcon={<Icon.Range />}>
            Copy time range as ISO
          </Core.Item>
          <Core.Item itemKey="python" startIcon={<Icon.Python />}>
            Copy time range as Python
          </Core.Item>
          <Core.Item itemKey="typescript" startIcon={<Icon.Typescript />}>
            Copy time range as Typescript
          </Core.Item>
        </>
      )}
    </Core.Menu>
  );
};
