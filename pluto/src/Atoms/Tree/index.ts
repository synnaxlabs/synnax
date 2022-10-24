import CoreTree from "./Tree";
export type { TreeProps, TreeEntry } from "./Tree";

type CoreTreeType = typeof CoreTree;

interface TreeType extends CoreTreeType {}

export const Tree = CoreTree as TreeType;
