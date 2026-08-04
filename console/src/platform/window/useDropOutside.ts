// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { Haul, useAsyncEffect, useSyncedRef } from "@synnaxlabs/pluto";
import { box, runtime, xy } from "@synnaxlabs/x";
import { listen } from "@tauri-apps/api/event";
import { useCallback, useEffect } from "react";
import { useStore } from "react-redux";

import { Session } from "@/session";

/** Called when a haul ends over the desktop, with the screen position it ended at. */
export interface OnDropOutside {
  (state: Haul.DraggingState, cursor: xy.XY): void;
}

export interface UseDropOutsideProps {
  canDrop: Haul.CanDrop;
  onDrop: OnDropOutside;
}

const TARGET: Haul.Item = { key: "drop-outside", type: "window" };

const useWindowsContain = (): ((cursor: xy.XY) => boolean) => {
  const store = useStore<Drift.StoreState>();
  return useCallback(
    (cursor) =>
      Drift.selectWindows(store.getState()).some(
        ({ stage, reserved, position, size }) =>
          stage === "created" &&
          reserved &&
          box.contains(box.construct(position ?? xy.ZERO, size), cursor),
      ),
    [store],
  );
};

// A drag that started in another window never called start() on this window's haul
// provider, so its local dragging ref is empty. Only the source window acts.
const useResolve = ({ canDrop, onDrop }: UseDropOutsideProps) => {
  const windowsContain = useWindowsContain();
  const propsRef = useSyncedRef({ canDrop, onDrop });
  return useCallback(
    (state: Haul.DraggingState, cursor: xy.XY): Haul.DropProps | null => {
      const { canDrop, onDrop } = propsRef.current;
      if (state.items.length === 0 || !canDrop(state) || windowsContain(cursor))
        return null;
      onDrop(state, cursor);
      return { target: TARGET, dropped: state.items };
    },
    [windowsContain],
  );
};

// macOS swallows the dragend of a drag released outside the window, so the drop is
// resolved from a native mouse_up event carrying screen coordinates instead. The event
// reaches every window; the empty-ref check above keeps the others out of it.
const useMacOS = (props: UseDropOutsideProps): void => {
  const ctx = Haul.useContext();
  const dragging = Haul.useDraggingRef();
  const resolve = useResolve(props);
  useAsyncEffect(async () => {
    if (Session.Runtime.ENGINE !== "tauri" || ctx == null) return;
    return await listen(
      "mouse_up",
      ({ payload: [x, y] }: { payload: [number, number] }) => {
        const dropped = resolve(dragging.current, xy.construct(x, y));
        if (dropped != null) ctx.drop(dropped);
      },
    );
  }, [ctx, resolve]);
};

const useWindows = (props: UseDropOutsideProps): void => {
  const ctx = Haul.useContext();
  const resolve = useResolve(props);
  useEffect(() => ctx?.bind(resolve), [ctx, resolve]);
};

/**
 * useDropOutside handles hauls released over the desktop rather than over any window,
 * the gesture that tears content into a new window. Drops landing inside another window
 * are that window's business and are left alone.
 */
export const useDropOutside = runtime.getOS() === "macOS" ? useMacOS : useWindows;
