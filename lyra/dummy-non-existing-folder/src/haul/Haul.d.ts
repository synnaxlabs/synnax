import "./Haul.css";
import { type destructor, type optional, type record, xy } from "@synnaxlabs/x";
import React, { type DragEvent, type DragEventHandler, type PropsWithChildren, type ReactNode, type RefObject } from "react";
import { z } from "zod";
import { type state } from "../state";
/** Zod schema for a draggable/droppable {@link Item}. */
export declare const itemZ: z.ZodObject<{
    key: z.ZodUnion<[z.ZodString, z.ZodNumber]>;
    type: z.ZodString;
    elementID: z.ZodOptional<z.ZodString>;
    data: z.ZodOptional<z.ZodUnknown>;
}, z.core.$strip>;
/**
 * A draggable/droppable payload identified by `type` and `key`. `data` is
 * required unless `Data` includes `undefined`.
 */
export type Item<Type extends string = string, Key extends record.Key = record.Key, Data = unknown> = {
    key: Key;
    type: Type;
    elementID?: string;
} & (undefined extends Data ? {
    data?: Data;
} : {
    data: Data;
});
/** Zod schema for {@link DraggingState}. */
export declare const draggingStateZ: z.ZodObject<{
    source: z.ZodObject<{
        key: z.ZodUnion<[z.ZodString, z.ZodNumber]>;
        type: z.ZodString;
        elementID: z.ZodOptional<z.ZodString>;
        data: z.ZodOptional<z.ZodUnknown>;
    }, z.core.$strip>;
    items: z.ZodArray<z.ZodObject<{
        key: z.ZodUnion<[z.ZodString, z.ZodNumber]>;
        type: z.ZodString;
        elementID: z.ZodOptional<z.ZodString>;
        data: z.ZodOptional<z.ZodUnknown>;
    }, z.core.$strip>>;
}, z.core.$strip>;
/** The item a drag originated from (`source`) and everything hauled with it. */
export interface DraggingState {
    source: Item;
    items: Item[];
}
/** Empty {@link Item}, used as a zero value. */
export declare const ZERO_ITEM: Item;
/** Empty {@link DraggingState}, used as a zero value. */
export declare const ZERO_DRAGGING_STATE: DraggingState;
/** The drop target and the items that landed on it. */
export interface DropProps {
    target: Item;
    dropped: Item[];
}
/** Item type used to represent an OS file drag. */
export declare const FILE_TYPE = "file";
/** Sentinel {@link Item} representing an OS file drag. */
export declare const FILE: Item;
/** Reports whether `event` is an OS file drag rather than an internal haul drag. */
export declare const isFileDrag: (event: DragEvent, dragging: DraggingState) => boolean;
/** Called on drag end; returns drop props to finalize, or null to ignore. */
interface DragEndInterceptor {
    (state: DraggingState, cursor: xy.XY): DropProps | null;
}
/**
 * Called once per drag when it resolves. `landed` reports whether the items reached a
 * drop target, whether an ordinary one or an interceptor's.
 */
export interface OnResolve {
    (landed: boolean): void;
}
/**
 * Haul drag-and-drop controls: the imperative drag/drop callbacks plus a live-state
 * ref. Its identity is stable across drags, so subscribing to it does not re-render
 * on drag-state changes. Read the live dragging state via {@link useDraggingState}
 * (re-renders) or {@link useDraggingRef} (does not).
 */
export interface ContextValue {
    /** Begins a drag of `items` originating from `source`. */
    start: (source: Item, items: Item[], onSuccessfulDrop?: (props: OnSuccessfulDropProps) => void) => void;
    /** Ends the drag at `cursor`, running bound interceptors to resolve any drop. */
    end: (cursor: xy.XY) => void;
    /** Finalizes a drop, invoking the drag's `onSuccessfulDrop` callback. */
    drop: (props: DropProps) => void;
    /** Registers an interceptor `end` consults to resolve drops outside a drop target. */
    bind: (interceptor: DragEndInterceptor) => destructor.Destructor;
    /**
     * Registers a watcher run once per drag when it resolves. A watcher cannot claim a
     * drop, so every watcher runs whatever an interceptor did and in any order.
     */
    watch: (onResolve: OnResolve) => destructor.Destructor;
    claimDragEvent: (event: Event) => boolean;
    /** A live ref to the current dragging state, readable at event time. */
    stateRef: RefObject<DraggingState>;
}
declare const useContext: () => ContextValue | null;
/**
 * Accesses the ambient drag/drop controls; null if no {@link Provider} is mounted.
 * The controls are identity-stable, so consuming them does not re-render on drag-state
 * changes.
 */
export { useContext };
/** Props for {@link Provider}. */
export interface ProviderProps extends PropsWithChildren {
    /** State hook backing the drag; defaults to `React.useState`. */
    useState?: state.PureUse<DraggingState>;
}
/** Provides haul drag-and-drop state to descendants. */
export declare const Provider: React.MemoExoticComponent<({ children, useState }: ProviderProps) => ReactNode>;
/** Returns a ref tracking the live {@link DraggingState} without re-rendering. */
export declare const useDraggingRef: () => RefObject<DraggingState>;
/** Returns the current {@link DraggingState}, re-rendering on change. */
export declare const useDraggingState: () => DraggingState;
/**
 * useOnResolve runs `onResolve` once per drag when it resolves, told whether the drag
 * landed on a drop target. Prefer it to watching {@link useDraggingState} clear: that
 * looks identical for a drop and for a cancelled drag, and it re-renders on every drag
 * in the window. `onResolve` need not be stable; the latest one always runs.
 */
export declare const useOnResolve: (onResolve: OnResolve) => void;
/** Passed to a drag's `onSuccessfulDrop` callback once a drop completes. */
export interface OnSuccessfulDropProps {
    target: Item;
    hauled: Item[];
    dropped: Item[];
}
/** Props for {@link useDrag}; `key` defaults to a generated ID. */
export interface UseDragProps extends optional.Optional<Item, "key"> {
}
/** Return value of {@link useDrag}. */
export interface UseDragReturn {
    /** Begins hauling `items`, invoking `onSuccessfulDrop` if they're dropped. */
    startDrag: (items: Item[], onSuccessfulDrop?: (props: OnSuccessfulDropProps) => void) => void;
    /** Native `dragend`/`mouseup` handler that resolves the drag. */
    onDragEnd: (e: DragEvent | MouseEvent) => void;
}
/** Wires an element up as a drag source of `type`. */
export declare const useDrag: ({ type, key }: UseDragProps) => UseDragReturn;
/** Predicate deciding whether a drop target accepts the current drag. */
export interface CanDrop {
    (state: DraggingState): boolean;
}
/** The {@link DraggingState} at drop time, plus the originating native event. */
export interface OnDropProps extends DraggingState {
    event?: DragEvent;
}
/** Handles a drop, returning the items actually consumed. */
export interface OnDrop {
    (props: OnDropProps): Item[];
}
/** Props passed to a drop target's `onDragOver` callback. */
export interface OnDragOverProps extends OnDropProps {
}
/** Props for {@link useDrop}; `key` defaults to a generated ID. */
export interface UseDropProps extends optional.Optional<Item, "key"> {
    canDrop: CanDrop;
    onDrop: OnDrop;
    onDragOver?: (props: OnDragOverProps) => void;
}
/** Return value of {@link useDrop}: DOM handlers to spread onto a drop target. */
export interface UseDropReturn {
    onDragOver: DragEventHandler;
    onDrop: DragEventHandler;
}
/** Wires an element up as a drop target of `type`. */
export declare const useDrop: ({ type, key, canDrop, onDrop, onDragOver, }: UseDropProps) => UseDropReturn;
/** Props for {@link useDragAndDrop}; `type`/`key` serve as both source and target. */
export interface UseDragAndDropProps extends Omit<UseDragProps, "source">, Omit<UseDropProps, "target">, optional.Optional<Item, "key"> {
}
/** Return value of {@link useDragAndDrop}. */
export interface UseDragAndDropReturn extends UseDragReturn, UseDropReturn {
}
/** Wires an element up as both a drag source and a drop target of `type`. */
export declare const useDragAndDrop: ({ type, key, ...rest }: UseDragAndDropProps) => UseDragAndDropReturn;
/** {@link CanDrop} matching any drag containing at least one item of `type`. */
export declare const canDropOfType: <I extends Item = Item>(type: I["type"]) => CanDrop;
/** Returns the subset of `entities` whose type is `type`. */
export declare const filterByType: (type: string, entities: Item[]) => Item[];
/** Memoizes `fn`, pre-filtering its `items` to `type` before it runs. */
export declare const useFilterByTypeCallback: (type: string, fn: OnDrop, deps: unknown[]) => OnDrop;
/** Props for a drop handled outside the normal tree; `onDrop` also gets the cursor. */
export interface UseDropOutsideProps extends Omit<UseDropProps, "onDrop"> {
    onDrop: (props: OnDropProps, cursor: xy.XY) => Item[];
}
//# sourceMappingURL=Haul.d.ts.map