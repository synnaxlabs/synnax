// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback } from "react";

import { Logo } from "@synnaxlabs/media";
import { Mosaic as PlutoMosaic, useDebouncedCallback } from "@synnaxlabs/pluto";
import type { MosaicProps } from "@synnaxlabs/pluto";
import type { Location } from "@synnaxlabs/x";
import { useDispatch } from "react-redux";

import { useLayoutPlacer } from "../hooks";
import {
  useSelectMosaic,
  moveLayoutMosaicTab,
  deleteLayoutMosaicTab,
  selectLayoutMosaicTab,
  resizeLayoutMosaicTab,
  renameLayoutMosaicTab,
} from "../store";

import { LayoutContent } from "./LayoutContent";

import { createLineVis } from "@/features/vis/components/line/types";

const emptyContent = <Logo.Watermark />;

/** LayoutMosaic renders the central layout mosaic of the application. */
export const LayoutMosaic = (): JSX.Element => {
  const dispatch = useDispatch();
  const mosaic = useSelectMosaic();
  const placer = useLayoutPlacer();

  const handleDrop = useCallback(
    (key: number, tabKey: string, loc: Location): void => {
      dispatch(moveLayoutMosaicTab({ key, tabKey, loc }));
    },
    [dispatch]
  );

  const handleClose = useCallback(
    (tabKey: string): void => {
      dispatch(deleteLayoutMosaicTab({ tabKey }));
    },
    [dispatch]
  );

  const handleSelect = useCallback(
    (tabKey: string): void => {
      dispatch(selectLayoutMosaicTab({ tabKey }));
    },
    [dispatch]
  );

  const handleRename = useCallback(
    (tabKey: string, name: string): void => {
      dispatch(renameLayoutMosaicTab({ tabKey, name }));
    },
    [dispatch]
  );

  const handleResize: MosaicProps["onResize"] = useDebouncedCallback(
    (key, size) => {
      dispatch(resizeLayoutMosaicTab({ key, size }));
    },
    100,
    [dispatch]
  );

  const handleCreate: MosaicProps["onCreate"] = useCallback(
    (mosaicKey) => {
      placer(
        createLineVis({
          tab: {
            mosaicKey,
          },
        })
      );
    },
    [placer]
  );

  return (
    <PlutoMosaic
      root={mosaic}
      onDrop={handleDrop}
      onClose={handleClose}
      onSelect={handleSelect}
      onResize={handleResize}
      emptyContent={emptyContent}
      onRename={handleRename}
      onCreate={handleCreate}
      size="small"
    >
      {(tab) => <LayoutContent layoutKey={tab.tabKey} />}
    </PlutoMosaic>
  );
};
