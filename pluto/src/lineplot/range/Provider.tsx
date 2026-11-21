// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/lineplot/range/Provider.css";

import type { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { useSyncedRef, useUniqueKey } from "@synnaxlabs/lyra/hooks";
import { Menu } from "@synnaxlabs/lyra/menu";
import { xy } from "@synnaxlabs/x";
import { type ReactElement, useEffect } from "react";

import { Aether } from "@/aether";
import { CSS } from "@/css";
import { useContext, useGridEntry } from "@/lineplot/Frame";
import { range } from "@/lineplot/range/aether";
import { Canvas } from "@/vis/canvas";

/** The props of the menu of a range in the strip. */
export interface MenuProps extends range.SelectedState {
  /** Whether the plot is zoomed to the range. */
  zoomed: boolean;
  /** Zooms the plot to the range, or back to the view before it when zoomed. */
  onZoom: () => void;
}

export interface ProviderProps extends Aether.ComponentProps {
  visible?: boolean;
  onHasAnnotationsChange?: (hasAnnotations: boolean) => void;
  menu?: Component.RenderProp<MenuProps>;
}

export const Provider = ({
  aetherKey,
  menu,
  visible = true,
  onHasAnnotationsChange,
  ...rest
}: ProviderProps): ReactElement => {
  const cKey = useUniqueKey(aetherKey);
  const { toggleZoom, zoomedKey } = useContext("Range.Provider");
  const [, { hovered, height }, setState] = Aether.use({
    aetherKey: cKey,
    type: range.Provider.TYPE,
    schema: range.providerStateZ,
    initialState: {
      ...rest,
      visible,
      cursor: null,
      menuOpen: false,
      hovered: null,
      count: 0,
      height: 0,
    },
    onAetherChange: ({ count }: range.ProviderState) =>
      onHasAnnotationsChange?.(count > 0),
  });
  useEffect(() => {
    setState((s) => ({ ...s, visible }));
  }, [visible, setState]);

  const gridStyle = useGridEntry(
    { key: cKey, loc: "top", size: visible ? height : 0, order: 3 },
    "Range.Provider",
  );

  const menuProps = Menu.useContextMenu();
  const visibleRef = useSyncedRef(menuProps.visible);
  useEffect(() => {
    setState((s) => ({ ...s, menuOpen: menuProps.visible }));
  }, [menuProps.visible, setState]);

  const handleMouseMove: React.MouseEventHandler<HTMLDivElement> = (e) => {
    const topLeft = Canvas.topLeft();
    if (topLeft == null) return;
    const cursor = xy.sub(xy.construct(e), topLeft);
    setState((state) => ({ ...state, cursor }));
  };

  const handleMouseLeave: React.MouseEventHandler<HTMLDivElement> = () => {
    if (!visibleRef.current) setState((state) => ({ ...state, cursor: null }));
  };

  return (
    <Menu.ContextMenu
      {...menuProps}
      menu={() => {
        if (menu == null || hovered == null) return null;
        return menu({
          ...hovered,
          zoomed: zoomedKey === hovered.key,
          onZoom: () => toggleZoom(hovered.key, hovered.viewport),
        });
      }}
    >
      <Flex.Box
        style={gridStyle}
        onContextMenu={menuProps.open}
        className={CSS.cls(
          CSS.B("lineplot-range-provider"),
          hovered != null && CSS.M("hovered"),
          menuProps.className,
        )}
        onClick={() => {
          if (hovered != null) toggleZoom(hovered.key, hovered.viewport);
        }}
        onMouseMove={handleMouseMove}
        onMouseLeave={handleMouseLeave}
      />
    </Menu.ContextMenu>
  );
};
