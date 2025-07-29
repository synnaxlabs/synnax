// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { box, xy } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { Aether } from "@/aether";
import { Align } from "@/align";
import { Annotation } from "@/annotation";
import { type RenderProp } from "@/component/renderProp";
import { useSyncedRef } from "@/hooks";
import { useUniqueKey } from "@/hooks/useUniqueKey";
import { Menu } from "@/menu";
import { annotation } from "@/vis/lineplot/annotation/aether";
import { useContext, useGridEntry } from "@/vis/lineplot/LinePlot";

export interface ProviderProps extends Aether.ComponentProps {
  parent: ontology.ID;
  menu?: RenderProp<annotation.SelectedState>;
}

export const Provider = ({
  aetherKey,
  parent,
  menu,
  ...rest
}: ProviderProps): ReactElement => {
  const cKey = useUniqueKey(aetherKey);
  const { setViewport, setHold } = useContext("Annotation.Provider");
  const [, { hovered, count }, setState] = Aether.use({
    aetherKey: cKey,
    type: annotation.Provider.TYPE,
    schema: annotation.providerStateZ,
    initialState: { ...rest, cursor: null, hovered: null, count: 0, parent },
  });
  const gridStyle = useGridEntry(
    { key: cKey, loc: "top", size: count > 0 ? 32 : 0, order: 4 },
    "Annotation.Provider",
  );

  const menuProps = Menu.useContextMenu();
  const visibleRef = useSyncedRef(menuProps.visible);

  const handleMouseMove: React.MouseEventHandler<HTMLDivElement> = useCallback(
    (e) =>
      setState((state: annotation.ProviderState) => ({
        ...state,
        cursor: { x: e.clientX, y: e.clientY },
      })),
    [setState],
  );

  const handleMouseLeave: React.MouseEventHandler<HTMLDivElement> = useCallback(() => {
    if (!visibleRef.current)
      setState((state: annotation.ProviderState) => ({ ...state, cursor: null }));
  }, [setState, visibleRef]);

  return (
    <Menu.ContextMenu
      {...menuProps}
      menu={() => {
        if (menu == null) return null;
        if (hovered != null) return menu(hovered);
        return <Annotation.CreateIcon />;
      }}
    >
      <Align.Space
        style={{
          ...gridStyle,
          cursor: hovered != null ? "pointer" : "crosshair",
          width: "100%",
          height: "100%",
        }}
        onContextMenu={menuProps.open}
        className={menuProps.className}
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
    </Menu.ContextMenu>
  );
};
