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
