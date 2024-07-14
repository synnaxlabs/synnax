// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layouts/Mosaic.css";

import { ontology } from "@synnaxlabs/client";
import { Logo } from "@synnaxlabs/media";
import {
  Eraser,
  Mosaic as Core,
  Nav,
  Synnax,
  useDebouncedCallback,
} from "@synnaxlabs/pluto";
import { type location } from "@synnaxlabs/x";
import { memo, type ReactElement, useCallback, useLayoutEffect } from "react";
import { useDispatch, useStore } from "react-redux";

import { NAV_DRAWERS, NavDrawer, NavMenu } from "@/components/nav/Nav";
import { Layout } from "@/layout";
import { Content } from "@/layout/Content";
import { usePlacer } from "@/layout/hooks";
import { useSelectActiveMosaicTabKey, useSelectMosaic } from "@/layout/selectors";
import {
  moveMosaicTab,
  remove,
  rename,
  resizeMosaicTab,
  selectMosaicTab,
  setNavDrawer,
} from "@/layout/slice";
import { createSelector } from "@/layouts/Selector";
import { LinePlot } from "@/lineplot";
import { SchematicServices } from "@/schematic/services";
import { SERVICES } from "@/services";
import { type RootStore } from "@/store";
import { Workspace } from "@/workspace";

const EmptyContent = (): ReactElement => (
  <Eraser.Eraser>
    <Logo.Watermark />;
  </Eraser.Eraser>
);

const emptyContent = <EmptyContent />;

export const MOSAIC_TYPE = "mosaic";

/** LayoutMosaic renders the central layout mosaic of the application. */
export const Mosaic = memo((): ReactElement => {
  const [windowKey, mosaic] = useSelectMosaic();
  const store = useStore();
  const activeTab = useSelectActiveMosaicTabKey();
  const client = Synnax.use();
  const placer = usePlacer();
  const dispatch = useDispatch();

  const handleDrop = useCallback(
    (key: number, tabKey: string, loc: location.Location): void => {
      dispatch(moveMosaicTab({ key, tabKey, loc, windowKey }));
    },
    [dispatch, windowKey],
  );

  const handleCreate = useCallback(
    (mosaicKey: number, location: location.Location, tabKeys?: string[]) => {
      if (tabKeys == null) {
        placer(
          createSelector({
            tab: { mosaicKey, location },
            location: "mosaic",
          }),
        );
        return;
      }
      tabKeys.forEach((tabKey) => {
        const res = ontology.stringIDZ.safeParse(tabKey);
        if (res.success) {
          const id = new ontology.ID(res.data);
          if (client == null) return;
          SERVICES[id.type].onMosaicDrop?.({
            client,
            store: store as RootStore,
            id,
            nodeKey: mosaicKey,
            location,
            placeLayout: placer,
          });
        } else
          placer(
            createSelector({
              tab: { mosaicKey, location },
              location: "mosaic",
            }),
          );
      });
    },
    [placer, store, client],
  );

  LinePlot.useTriggerHold({
    defaultMode: "hold",
    hold: [["H"]],
    toggle: [["H", "H"]],
  });

  const handleClose = useCallback(
    (tabKey: string): void => {
      dispatch(remove({ keys: [tabKey] }));
    },
    [dispatch],
  );

  const handleSelect = useCallback(
    (tabKey: string): void => {
      dispatch(selectMosaicTab({ tabKey }));
    },
    [dispatch],
  );

  const handleRename = useCallback(
    (tabKey: string, name: string): void => {
      dispatch(rename({ key: tabKey, name }));
    },
    [dispatch],
  );

  const handleResize = useDebouncedCallback(
    (key, size) => {
      dispatch(resizeMosaicTab({ key, size, windowKey }));
    },
    100,
    [dispatch, windowKey],
  );

  const workspaceKey = Workspace.useSelectActiveKey();

  const handleFileDrop = useCallback(
    (nodeKey: number, event: React.DragEvent<HTMLDivElement>) => {
      const files = Array.from(event.dataTransfer.files);
      if (files.length === 0) return;
      files.forEach((file) => {
        if (file.type !== "application/json") return;
        file
          ?.arrayBuffer()
          .then((b) => {
            const fileAsJSON = JSON.parse(new TextDecoder().decode(b));
            const name = file.name.slice(0, -5);
            SchematicServices.FileHandler({
              nodeKey,
              file: fileAsJSON,
              placer,
              name,
              client,
              workspaceKey,
            });
          })
          .catch((e) => {
            console.error(e);
          });
      });
    },
    [dispatch],
  );

  return (
    <Core.Mosaic
      root={mosaic}
      onDrop={handleDrop}
      onClose={handleClose}
      onSelect={handleSelect}
      onResize={handleResize}
      emptyContent={emptyContent}
      onRename={handleRename}
      onCreate={handleCreate}
      activeTab={activeTab ?? undefined}
      onFileDrop={handleFileDrop}
    >
      {({ tabKey }) => <Content key={tabKey} layoutKey={tabKey} />}
    </Core.Mosaic>
  );
});
Mosaic.displayName = "Mosaic";

export const Window = memo(({ layoutKey }: Layout.RendererProps): ReactElement => {
  const { menuItems, onSelect } = Layout.useNavDrawer("bottom", NAV_DRAWERS);
  const d = useDispatch();
  useLayoutEffect(() => {
    d(
      setNavDrawer({
        windowKey: layoutKey,
        location: "bottom",
        menuItems: ["visualization"],
        activeItem: "visualization",
      }),
    );
  }, [layoutKey]);
  return (
    <>
      <Mosaic />
      <NavDrawer location="bottom" />
      <Nav.Bar
        className="console-main-nav"
        location="bottom"
        style={{ paddingRight: "1.5rem" }}
        size={7 * 6}
      >
        <Nav.Bar.End>
          <NavMenu onChange={onSelect}>{menuItems}</NavMenu>
        </Nav.Bar.End>
      </Nav.Bar>
    </>
  );
});
Window.displayName = "MosaicWindow";

export const FILE_HANDLERS = [SchematicServices.FileHandler];
