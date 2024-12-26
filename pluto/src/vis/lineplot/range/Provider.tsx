// Copyright 2024 Synnax Labs, Inc.
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
import { Align } from "@/align";
import { useSyncedRef } from "@/hooks";
import { Menu } from "@/menu";
import { type RenderProp } from "@/util/renderProp";
import { useContext, useGridEntry } from "@/vis/lineplot/LinePlot";
import { range } from "@/vis/lineplot/range/aether";

export interface ProviderProps {
  menu?: RenderProp<range.SelectedState>;
}

export const Provider = Aether.wrap<ProviderProps>(
  "Annotation.Provider",
  ({ aetherKey, menu, ...props }): ReactElement => {
    const { setViewport, setHold } = useContext("Range.Provider");
    const [, { hovered, count }, setState] = Aether.use({
      aetherKey,
      type: range.Provider.TYPE,
      schema: range.providerStateZ,
      initialState: {
        ...props,
        cursor: null,
        hovered: null,
        count: 0,
      },
    });
    const gridStyle = useGridEntry(
      {
        key: aetherKey,
        loc: "top",
        size: count > 0 ? 32 : 0,
        order: 3,
      },
      "Annotation.Provider",
    );

    const menuProps = Menu.useContextMenu();
    const visibleRef = useSyncedRef(menuProps.visible);

    const handleMouseEnter: React.MouseEventHandler<HTMLDivElement> = useCallback(
      (e) => {
        // add an event listener for the movement until it leaves
        const handleMouseMove = (e: MouseEvent) => {
          setState((state) => ({ ...state, cursor: { x: e.clientX, y: e.clientY } }));
        };
        const target = e.currentTarget;
        target.addEventListener("mousemove", handleMouseMove);
        target.addEventListener(
          "mouseleave",
          () => {
            target.removeEventListener("mousemove", handleMouseMove);
            if (!visibleRef.current) setState((state) => ({ ...state, cursor: null }));
          },
          { once: true },
        );
      },
      [setState],
    );

    return (
      <Menu.ContextMenu
        style={{
          ...gridStyle,
          cursor: hovered != null ? "pointer" : "default",
        }}
        {...menuProps}
        menu={() => {
          if (menu == null || hovered == null) return null;
          return menu(hovered);
        }}
      >
        <Align.Space
          style={{ width: "100%", height: "100%" }}
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
          onMouseEnter={handleMouseEnter}
        />
      </Menu.ContextMenu>
    );
  },
);
