// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, memo, useCallback } from "react";

import { Logo } from "@synnaxlabs/media";
import { Mosaic as Core, useDebouncedCallback } from "@synnaxlabs/pluto";
import { type location } from "@synnaxlabs/x";
import { useDispatch } from "react-redux";

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
import { Vis } from "@/vis";

const emptyContent = <Logo.Watermark />;

/** LayoutMosaic renders the central layout mosaic of the application. */
export const Mosaic = memo((): ReactElement => {
  const dispatch = useDispatch();
  const [windowKey, mosaic] = useSelectMosaic();
  const placer = usePlacer();

  const handleDrop = useCallback(
    (key: number, tabKey: string, loc: location.Crude): void => {
      dispatch(moveMosaicTab({ key, tabKey, loc, windowKey }));
    },
    [dispatch, windowKey]
  );

  LinePlot.useTriggerHold({
    defaultMode: "hold",
    hold: [["H"]],
    toggle: [["H", "H"]],
  });

  const handleClose = useCallback(
    (tabKey: string): void => {
      dispatch(remove(tabKey));
    },
    [dispatch]
  );

  const handleSelect = useCallback(
    (tabKey: string): void => {
      dispatch(selectMosaicTab({ tabKey }));
    },
    [dispatch]
  );

  const handleRename = useCallback(
    (tabKey: string, name: string): void => {
      dispatch(rename({ key: tabKey, name }));
    },
    [dispatch]
  );

  const handleResize = useDebouncedCallback(
    (key, size) => {
      dispatch(resizeMosaicTab({ key, size, windowKey }));
    },
    100,
    [dispatch, windowKey]
  );

  const handleCreate = useCallback(
    (mosaicKey: number) => {
      placer(Vis.create({ tab: { mosaicKey } }));
    },
    [placer]
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
