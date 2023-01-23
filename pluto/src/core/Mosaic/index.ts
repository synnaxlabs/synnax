// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Mosaic as CoreMosaic } from "./Mosaic";
import {
  insertMosaicTab,
  moveMosaicTab,
  removeMosaicTab,
  resizeMosaicLeaf,
  selectMosaicTab,
  renameMosaicTab,
} from "./mosaicTree";
import { useMosaic } from "./useMosaic";
export * from "./types";

type CoreMosaicType = typeof CoreMosaic;

export interface MosaicType extends CoreMosaicType {
  use: typeof useMosaic;
  insertTab: typeof insertMosaicTab;
  removeTab: typeof removeMosaicTab;
  selectTab: typeof selectMosaicTab;
  moveTab: typeof moveMosaicTab;
  resizeLeaf: typeof resizeMosaicLeaf;
  renameTab: typeof renameMosaicTab;
}

export const Mosaic = CoreMosaic as MosaicType;

Mosaic.use = useMosaic;
Mosaic.insertTab = insertMosaicTab;
Mosaic.removeTab = removeMosaicTab;
Mosaic.selectTab = selectMosaicTab;
Mosaic.moveTab = moveMosaicTab;
Mosaic.resizeLeaf = resizeMosaicLeaf;
Mosaic.renameTab = renameMosaicTab;
