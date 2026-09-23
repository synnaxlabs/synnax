import { type record, xy } from "@synnaxlabs/x";
import { type RefObject } from "react";
import { type ClipboardHandler } from "./Diagram";
/** ClipboardNode is the minimal shape the clipboard needs from a diagram node. */
export interface ClipboardNode {
    key: string;
    position: xy.XY;
}
/** ClipboardEdge is the minimal shape the clipboard needs from a diagram edge. */
export interface ClipboardEdge {
    source: {
        node: string;
    };
    target: {
        node: string;
    };
}
/** Snapshot is the current set of nodes, edges, and configs for a diagram. */
export interface Snapshot<N extends ClipboardNode, E extends ClipboardEdge, C = record.Unknown> {
    nodes: N[];
    edges: E[];
    configs: Record<string, C>;
}
/** PastedNode is a node remapped for paste, with its copied config attached. */
export interface PastedNode<N extends ClipboardNode, C = record.Unknown> {
    node: N;
    config: C | undefined;
}
/** PastedEdge is an edge remapped for paste, with its copied config attached. */
export interface PastedEdge<E extends ClipboardEdge, C = record.Unknown> {
    edge: E;
    config: C | undefined;
}
/**
 * PasteResult is the outcome of a paste, as plain data. Node keys are freshly
 * generated and positions are offset to the cursor; edge endpoints are remapped
 * onto the new node keys and edges whose endpoints did not survive are dropped.
 * It is up to the consumer to persist these values however it sees fit.
 */
export interface PasteResult<N extends ClipboardNode, E extends ClipboardEdge, C = record.Unknown> {
    nodes: PastedNode<N, C>[];
    edges: PastedEdge<E, C>[];
    /** remap maps copied node keys to their fresh replacements. */
    remap: Record<string, string>;
}
/** SelectionKeys identifies clipboard items by key, split by kind. */
export interface SelectionKeys {
    nodes: string[];
    edges: string[];
}
/**
 * ClipboardAdapter binds the generic cut/copy/paste mechanics to a diagram domain.
 * It deals only in plain diagram data: the adapter reads a snapshot to copy and
 * receives a paste result to apply. The diagram has no knowledge of how that data
 * is stored or persisted.
 */
export interface ClipboardAdapter<N extends ClipboardNode, E extends ClipboardEdge, C = record.Unknown> {
    /** mime is the clipboard MIME type used to read and write the payload. */
    mime: string;
    /**
     * edgeKey returns an edge's identity, used for selection membership and for
     * associating an edge with its copied config.
     */
    edgeKey: (edge: E) => string;
    /** getSnapshot returns the current diagram contents, or null if unavailable. */
    getSnapshot: () => Snapshot<N, E, C> | null;
    /** apply persists a paste result. Called only when at least one item pasted. */
    apply: (result: PasteResult<N, E, C>) => void;
    /** remove deletes the given items from the diagram. Called by cut after it copies. */
    remove: (keys: SelectionKeys) => void;
}
export interface UseClipboardParams<N extends ClipboardNode, E extends ClipboardEdge, C = record.Unknown> {
    adapter: ClipboardAdapter<N, E, C>;
    selected?: string[];
    /** onCut receives the selection that survives a cut. */
    onCut?: (remaining: string[]) => void;
    container?: RefObject<HTMLDivElement | null>;
}
export interface UseClipboardReturn {
    onCopy: ClipboardHandler;
    onCut: ClipboardHandler;
    onPaste: ClipboardHandler;
    copy: () => void;
    cut: () => void;
    paste: () => void;
}
/**
 * useClipboard provides copy, cut, and paste handlers for a node/edge diagram. It
 * owns the clipboard mechanics (MIME plumbing, version gating, anchor geometry, key
 * remapping); the adapter supplies the diagram data and persists the result.
 */
export declare const useClipboard: <N extends ClipboardNode, E extends ClipboardEdge, C>({ adapter, selected, onCut: onCutProp, container, }: UseClipboardParams<N, E, C>) => UseClipboardReturn;
//# sourceMappingURL=clipboard.d.ts.map