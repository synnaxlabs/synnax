// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback } from "react";

import type { Location, Tab } from "@synnaxlabs/pluto";
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

import { LayoutContent } from "./LayoutContent";

import { Logo } from "@/components";

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
    (tabKey: string, title: string): void => {
      dispatch(renameLayoutMosaicTab({ tabKey, title }));
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

  console.log(mosaic);

  return (
    <PlutoMosaic
      root={mosaic}
      onDrop={handleDrop}
      onClose={handleClose}
      onSelect={handleSelect}
      onResize={handleResize}
      emptyContent={emptyContent}
      onTitleChange={handleRename}
    >
      {LayoutMosaicContent}
    </PlutoMosaic>
  );
};

const LayoutMosaicContent = ({ tab }: { tab: Tab }): JSX.Element => (
  <LayoutContent layoutKey={tab.tabKey} />
);
