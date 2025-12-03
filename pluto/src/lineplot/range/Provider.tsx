// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, xy } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { Aether } from "@/aether";
import { type RenderProp } from "@/component/renderProp";
import { ContextMenu } from "@/context-menu";
import { Flex } from "@/flex";
import { useSyncedRef } from "@/hooks";
import { useUniqueKey } from "@/hooks/useUniqueKey";
import { useContext, useGridEntry } from "@/lineplot/LinePlot";
import { range } from "@/lineplot/range/aether";

export interface ProviderProps extends Aether.ComponentProps {
  menu?: RenderProp<range.SelectedState>;
}

export const Provider = ({ aetherKey, menu, ...rest }: ProviderProps): ReactElement => {
  const cKey = useUniqueKey(aetherKey);
  const { setViewport, setHold } = useContext("Range.Provider");
  const [, { hovered, count }, setState] = Aether.use({
    aetherKey: cKey,
    type: range.Provider.TYPE,
    schema: range.providerStateZ,
    initialState: { ...rest, cursor: null, hovered: null, count: 0 },
  });
  const gridStyle = useGridEntry(
    { key: cKey, loc: "top", size: count > 0 ? 32 : 0, order: 3 },
    "Range.Provider",
  );

  const contextMenuProps = ContextMenu.use();
  const visibleRef = useSyncedRef(contextMenuProps.visible);

  const handleMouseMove: React.MouseEventHandler<HTMLDivElement> = useCallback(
    (e) => setState((state) => ({ ...state, cursor: { x: e.clientX, y: e.clientY } })),
    [setState],
  );

  const handleMouseLeave: React.MouseEventHandler<HTMLDivElement> = useCallback(() => {
    if (!visibleRef.current) setState((state) => ({ ...state, cursor: null }));
  }, [setState, visibleRef]);

  return (
    <ContextMenu.ContextMenu
      {...contextMenuProps}
      menu={() => {
        if (menu == null || hovered == null) return null;
        return menu(hovered);
      }}
    >
      <Flex.Box
        style={{
          ...gridStyle,
          cursor: hovered != null ? "pointer" : "default",
          width: "100%",
          height: "100%",
        }}
        onContextMenu={contextMenuProps.open}
        className={contextMenuProps.className}
        onClick={() => {
          if (hovered != null) {
            setViewport({
              box: box.construct(
                { x: hovered.viewport.lower, y: 0 },
                { x: hovered.viewport.upper, y: 1 },
              ),
              mode: "zoom",
              cursor: xy.ZERO,
              stage: "start",
            });
            setHold(true);
          }
        }}
        onMouseMove={handleMouseMove}
        onMouseLeave={handleMouseLeave}
      />
    </ContextMenu.ContextMenu>
  );
};
