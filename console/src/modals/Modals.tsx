// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/modals/Modals.css";

import { type Icon } from "@synnaxlabs/pluto";
import { type dimensions, id } from "@synnaxlabs/x";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useContext,
  useMemo,
  useRef,
  useState,
} from "react";

import { Modal } from "@/modals/Modal";

/** Sizing and chrome configuration for the modal dialog. */
export interface WindowProps {
  resizable?: boolean;
  size?: dimensions.Dimensions;
  minSize?: dimensions.Dimensions;
  maxSize?: dimensions.Dimensions;
  navTop?: boolean;
  showTitle?: boolean;
}

/** A fully specified modal, ready to render. */
export interface Spec<A = unknown> {
  /** The unique key of the modal. */
  key: string;
  /** The registered renderer type for the modal. */
  type: string;
  /** The display name shown in the modal chrome. */
  name: string;
  /** Type-specific arguments handed to the renderer via {@link useArgs}. */
  args?: A;
  /** An optional icon shown alongside the name. */
  icon?: Icon.ReactElement | string;
  /** Sizing and chrome configuration. */
  window?: WindowProps;
}

/** A modal spec with an optional key; one is generated on open if omitted. */
export interface BaseState<A = unknown> extends Omit<Spec<A>, "key"> {
  key?: string;
}

/** Builds a {@link BaseState} dynamically at open time. */
export interface Creator<A = unknown> {
  (): BaseState<A>;
}

export type PlacerArgs<A = unknown> = BaseState<A> | Creator<A>;

/**
 * Opens a modal and resolves with its result, or null if dismissed. Only one modal is
 * open per window at a time; opening a new one replaces the current.
 */
export interface Placer {
  <R = unknown, A = unknown>(layout: PlacerArgs<A>): Promise<R | null>;
}

interface ContextValue {
  open: Placer;
  close: (result?: unknown) => void;
}

const Context = createContext<ContextValue | null>(null);
const ActiveContext = createContext<Spec | null>(null);

const useContextValue = (): ContextValue => {
  const ctx = useContext(Context);
  if (ctx == null) throw new Error("Modals.Provider is not mounted");
  return ctx;
};

/** Returns the opener for placing a modal. @see Placer */
export const usePlacer = (): Placer => useContextValue().open;

/** Returns a function that closes the active modal. */
export const useCloser = (): ((result?: unknown) => void) => useContextValue().close;

/** Returns the arguments of the active modal. */
export const useArgs = <A,>(): A => (useContext(ActiveContext)?.args ?? {}) as A;

export const Provider = ({ children }: PropsWithChildren): ReactElement => {
  const [spec, setSpec] = useState<Spec | null>(null);
  const resolve = useRef<((result: unknown) => void) | null>(null);

  const close = useCallback((result?: unknown) => {
    resolve.current?.(result ?? null);
    resolve.current = null;
    setSpec(null);
  }, []);

  const open = useCallback<Placer>((layout) => {
    resolve.current?.(null);
    const base = typeof layout === "function" ? layout() : layout;
    const next: Spec = { ...base, key: base.key ?? id.create() };
    return new Promise((res) => {
      resolve.current = res as (result: unknown) => void;
      setSpec(next);
    });
  }, []);

  const value = useMemo<ContextValue>(() => ({ open, close }), [open, close]);

  return (
    <Context value={value}>
      <ActiveContext value={spec}>
        {children}
        {spec != null && <Modal spec={spec} onClose={close} />}
      </ActiveContext>
    </Context>
  );
};
