// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, memo, useCallback } from "react";

import { ontology } from "@synnaxlabs/client";
import { Logo } from "@synnaxlabs/media";
import {
  Mosaic as Core,
  Eraser,
  Synnax,
  useDebouncedCallback,
} from "@synnaxlabs/pluto";
import { type location } from "@synnaxlabs/x";
import { useStore } from "react-redux";

import { useSyncerDispatch } from "@/hooks/dispatchers";
import { Content } from "@/layout/Content";
import { usePlacer } from "@/layout/hooks";
import { useSelectMosaic } from "@/layout/selectors";
import {
  moveMosaicTab,
  remove,
  rename,
  resizeMosaicTab,
  selectMosaicTab,
} from "@/layout/slice";
import { LinePlot } from "@/lineplot";
import { SERVICES } from "@/services";
import { type RootStore } from "@/store";
import { Vis } from "@/vis";
import { Workspace } from "@/workspace";

const EmptyContent = (): ReactElement => (
  <Eraser.Eraser>
    <Logo.Watermark />;
  </Eraser.Eraser>
);

const emptyContent = <EmptyContent />;

/** LayoutMosaic renders the central layout mosaic of the application. */
export const Mosaic = memo((): ReactElement => {
  const [windowKey, mosaic] = useSelectMosaic();

  const client = Synnax.use();
  const store = useStore();
  const placer = usePlacer();

  const syncer = Workspace.useLayoutSyncer();
  const dispatch = useSyncerDispatch(syncer, 1000);

  const handleDrop = useCallback(
    (key: number, tabKey: string, loc: location.Location): void => {
      dispatch(moveMosaicTab({ key, tabKey, loc, windowKey }));
    },
    [dispatch, windowKey],
  );

  const handleCreate = useCallback(
    (mosaicKey: number, location: location.Location, tabKeys?: string[]) => {
      if (tabKeys == null) {
        placer(Vis.create({ tab: { mosaicKey, location }, location: "mosaic" }));
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
        } else placer(Vis.create({ tab: { mosaicKey, location }, location: "mosaic" }));
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
      size="medium"
    >
      {(tab) => <Content key={tab.tabKey} layoutKey={tab.tabKey} />}
    </Core.Mosaic>
  );
});
Mosaic.displayName = "LayoutMosaic";
