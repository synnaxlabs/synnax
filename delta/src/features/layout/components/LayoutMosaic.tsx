// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Mosaic as PlutoMosaic, debounce } from "@synnaxlabs/pluto";
import type { Location, Tab } from "@synnaxlabs/pluto";
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

/** LayoutMosaic renders the central layout mosaic of the application. */
export const LayoutMosaic = (): JSX.Element => {
  const dispatch = useDispatch();
  const mosaic = useSelectMosaic();

  const handleDrop = (key: number, tabKey: string, loc: Location): void => {
    dispatch(moveLayoutMosaicTab({ key, tabKey, loc }));
  };

  const handleClose = (tabKey: string): void => {
    dispatch(deleteLayoutMosaicTab({ tabKey }));
  };

  const handleSelect = (tabKey: string): void => {
    dispatch(selectLayoutMosaicTab({ tabKey }));
  };

  const handleRename = (tabKey: string, title: string): void => {
    dispatch(renameLayoutMosaicTab({ tabKey, title }));
  };

  const handleResize = debounce(
    (key: number, size: number) => dispatch(resizeLayoutMosaicTab({ key, size })),
    0
  );

  return (
    <PlutoMosaic
      root={mosaic}
      onDrop={handleDrop}
      onClose={handleClose}
      onSelect={handleSelect}
      onResize={handleResize}
      emptyContent={Logo.Watermark}
      onTitleChange={handleRename}
    >
      {LayoutMosaicContent}
    </PlutoMosaic>
  );
};

const LayoutMosaicContent = ({ tab }: { tab: Tab }): JSX.Element => (
  <LayoutContent layoutKey={tab.tabKey} />
);
