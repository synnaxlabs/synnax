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
import { type ReactElement, useEffect, useRef } from "react";

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
  /** True when a click on the strip does not open a break, as on an axis with none. */
  breaksDisabled?: boolean;
  /** Called with whether the strip shows ranges, and with false when it unmounts. */
  onHasAnnotationsChange?: (hasAnnotations: boolean) => void;
  menu?: Component.RenderProp<MenuProps>;
}

export const Provider = ({
  aetherKey,
  menu,
  visible = true,
  breaksDisabled = false,
  onHasAnnotationsChange,
  ...rest
}: ProviderProps): ReactElement => {
  const cKey = useUniqueKey(aetherKey);
  const { toggleZoom, toggleBreak, zoomedKey } = useContext("Range.Provider");
  const reported = useRef(false);
  const onHasAnnotationsChangeRef = useSyncedRef(onHasAnnotationsChange);
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
    onAetherChange: ({ count }: range.ProviderState) => {
      reported.current = count > 0;
      onHasAnnotationsChange?.(count > 0);
    },
  });
  useEffect(
    () => () => {
      if (reported.current) onHasAnnotationsChangeRef.current?.(false);
    },
    [onHasAnnotationsChangeRef],
  );
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
        onClick={(e) => {
          // A break band runs through the strip, so a click on it opens the break.
          if (hovered != null) toggleZoom(hovered.key, hovered.viewport);
          else if (!breaksDisabled) toggleBreak(e.clientX, "plot");
        }}
        onMouseMove={handleMouseMove}
        onMouseLeave={handleMouseLeave}
      />
    </Menu.ContextMenu>
  );
};
