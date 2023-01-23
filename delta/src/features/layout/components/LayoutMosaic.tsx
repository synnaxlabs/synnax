// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback } from "react";

import type { Location } from "@synnaxlabs/pluto";
import { Mosaic as PlutoMosaic, useDebouncedCallback } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import {
  useSelectMosaic,
  moveLayoutMosaicTab,
  deleteLayoutMosaicTab,
  selectLayoutMosaicTab,
  resizeLayoutMosaicTab,
  renameLayoutMosaicTab,
} from "../store";

import { Logo } from "@/components";

import { LayoutContent } from "./LayoutContent";

const emptyContent = <Logo.Watermark />;

/** LayoutMosaic renders the central layout mosaic of the application. */
export const LayoutMosaic = (): JSX.Element => {
  const dispatch = useDispatch();
  const mosaic = useSelectMosaic();

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

  const handleResize = useDebouncedCallback(
    (key: number, size: number) => {
      dispatch(resizeLayoutMosaicTab({ key, size }));
    },
    100,
    [dispatch]
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
    >
      {(tab) => <LayoutContent layoutKey={tab.tabKey} />}
    </PlutoMosaic>
  );
};
