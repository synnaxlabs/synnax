import { Tree as CoreTree } from "./Tree";
export type { TreeProps, TreeLeaf } from "./Tree";

type CoreTreeType = typeof CoreTree;

interface TreeType extends CoreTreeType {}

export const Tree = CoreTree as TreeType;
