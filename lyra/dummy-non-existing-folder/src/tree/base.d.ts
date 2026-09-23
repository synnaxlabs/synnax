import { type compare, type record } from "@synnaxlabs/x";
export interface Node<K extends record.Key = string> {
    key: K;
    children?: Node<K>[];
}
export interface NodeShape {
    depth: number;
    expanded: boolean;
    hasChildren: boolean;
}
export interface Shape<K extends record.Key = string> {
    keys: K[];
    nodes: NodeShape[];
}
export declare const shouldExpand: <K extends record.Key = string>(node: Node<K>, expanded: K[]) => boolean;
export interface FlattenProps<K extends record.Key = string> {
    nodes: Node<K>[];
    expanded: K[];
    sort?: compare.Comparator<Node<K>>;
    depth?: number;
    path?: string;
}
export declare const flatten: <K extends record.Key = string>({ nodes, expanded, sort, depth, }: FlattenProps<K>) => Shape<K>;
export interface MoveNodeProps<K extends record.Key = string> {
    tree: Node<K>[];
    destination: K | null;
    keys: K | K[];
}
export declare const moveNode: <K extends record.Key = string>({ tree, destination, keys, }: MoveNodeProps<K>) => Node<K>[];
export interface RemoveNodeProps<K extends record.Key = string> {
    tree: Node<K>[];
    keys: K | K[];
    parent?: K | null;
}
export declare const removeNode: <K extends record.Key = string>({ tree, keys, parent, }: RemoveNodeProps<K>) => Node<K>[];
export interface SetNodeProps<K extends record.Key = string> {
    tree: Node<K>[];
    destination: K | null;
    additions: Node<K> | Node<K>[];
    throwOnMissing?: boolean;
}
export declare const setNode: <K extends record.Key = string>({ tree, destination, additions, throwOnMissing, }: SetNodeProps<K>) => Node<K>[];
export interface UpdateNodeProps<K extends record.Key = string> {
    tree: Node<K>[];
    key: K;
    updater: (node: Node<K>) => Node<K>;
    throwOnMissing?: boolean;
}
export declare const updateNode: <K extends record.Key = string>({ tree, key, updater, throwOnMissing, }: UpdateNodeProps<K>) => Node<K>[];
interface UpdateNodeChildren<K extends record.Key = string> {
    tree: Node<K>[];
    parent: K;
    updater: (nodes: Node<K>[]) => Node<K>[];
    throwOnMissing?: boolean;
}
export declare const updateNodeChildren: <K extends record.Key = string>({ tree, parent, updater, throwOnMissing, }: UpdateNodeChildren<K>) => Node<K>[];
export interface FindNodeProps<K extends record.Key = string> {
    tree: Node<K>[];
    key: K;
    depth?: number;
}
export declare const findNode: <K extends record.Key = string>({ tree, key, depth, }: FindNodeProps<K>) => Node<K> | null;
export interface FindNodesProps<K extends record.Key = string> {
    tree: Node<K>[];
    keys: K[];
}
export declare const findNodes: <K extends record.Key = string>({ tree, keys, }: FindNodesProps<K>) => Node<K>[];
export interface FindNodeParentProps<K extends record.Key = string> {
    tree: Node<K>[];
    key: K;
}
export declare const findNodeParent: <K extends record.Key = string>({ tree, key, }: FindNodeParentProps<K>) => Node<K> | null;
export declare const deepCopy: <K extends record.Key = string>(nodes: Node<K>[]) => Node<K>[];
export declare const getDescendants: <K extends record.Key = string>(...node: Node<K>[]) => Node<K>[];
export declare const filterShape: <K extends record.Key = string>(shape: Shape<K>, match: (key: K, depth: number) => boolean) => Shape<K>;
export declare const getAllNodesOfMinDepth: <K extends record.Key = string>(data: Shape<K>) => K[];
/**
 * Returns the depth of the node with the given key, or null if the shape has no such
 * key. A shape holds only visible rows, so a node under a contracted parent is absent
 * even though it is still in the tree.
 */
export declare const getDepth: <K extends record.Key = string>(key: K, shape: Shape<K>) => number | null;
/** Orders keys shallowest-first. Keys the shape does not contain sort last. */
export declare const compareDepth: <K extends record.Key = string>(shape: Shape<K>) => compare.Comparator<K>;
export declare const getNodeShape: <K extends record.Key = string>(shape: Shape<K>, key: K) => NodeShape | null;
export {};
//# sourceMappingURL=base.d.ts.map