// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { Haul } from "@synnaxlabs/lyra/haul";
import { useSyncedRef } from "@synnaxlabs/lyra/hooks";
import { Status } from "@synnaxlabs/lyra/status";
import { box, runtime, xy } from "@synnaxlabs/x";
import { invoke } from "@tauri-apps/api/core";
import { useEffect } from "react";
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
  return (cursor) =>
    Drift.selectWindows(store.getState()).some(
      ({ stage, reserved, position, size }) =>
        stage === "created" &&
        reserved &&
        box.contains(box.construct(position ?? xy.ZERO, size), cursor),
    );
};

// The drag's source window tears it out, so callers pass the drag this window started.
const useResolve = ({ canDrop, onDrop }: UseDropOutsideProps) => {
  const windowsContain = useWindowsContain();
  const propsRef = useSyncedRef({ canDrop, onDrop });
  return (state: Haul.DraggingState, cursor: xy.XY): Haul.DropProps | null => {
    const { canDrop, onDrop } = propsRef.current;
    if (state.items.length === 0 || !canDrop(state) || windowsContain(cursor))
      return null;
    onDrop(state, cursor);
    return { target: TARGET, dropped: state.items };
  };
};

// macOS swallows the dragend of a drag released outside the window, so each drag asks
// the native side for the cursor position at release. The native side polls only while
// the button is down.
const useMacOS = (props: UseDropOutsideProps): void => {
  const ctx = Haul.useContext();
  const resolveRef = useSyncedRef(useResolve(props));
  const handleError = Status.useErrorHandler();
  useEffect(() => {
    if (Session.Runtime.ENGINE !== "tauri" || ctx == null) return;
    const controller = new AbortController();
    const handleDragStart = () =>
      handleError(async () => {
        const [x, y] = await invoke<[number, number]>("mouse_release");
        if (controller.signal.aborted) return;
        const cursor = xy.construct(x, y);
        const dropped = resolveRef.current(ctx.startedRef.current, cursor);
        if (dropped != null) ctx.drop(dropped);
      }, "Failed to drop outside the window");
    window.addEventListener("dragstart", handleDragStart, {
      capture: true,
      signal: controller.signal,
    });
    return () => controller.abort();
  }, [ctx]);
};

// The browser has no desktop to release content over. Its runtime reports no window
// geometry, so every unclaimed drop would read as one.
const useWindows = (props: UseDropOutsideProps): void => {
  const ctx = Haul.useContext();
  const resolve = useResolve(props);
  useEffect(() => {
    if (Session.Runtime.ENGINE !== "tauri") return;
    return ctx?.bind(resolve);
  }, [ctx, resolve]);
};

/**
 * useDropOutside handles hauls released over the desktop rather than over any window,
 * the gesture that tears content into a new window. Drops landing inside another window
 * are that window's business and are left alone.
 */
export const useDropOutside = runtime.getOS() === "macOS" ? useMacOS : useWindows;
