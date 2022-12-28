import { Tab } from "@/atoms";
import { Direction } from "@/util/spatial";

export interface MosaicLeaf {
  key: number;
  tabs?: Tab[];
  direction?: Direction;
  first?: MosaicLeaf;
  last?: MosaicLeaf;
  selected?: string;
  size?: number;
}
