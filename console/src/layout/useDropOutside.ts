// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { Haul, Mosaic, Status, useAsyncEffect, useSyncedRef } from "@synnaxlabs/pluto";
import { box, runtime, xy } from "@synnaxlabs/x";
import { listen } from "@tauri-apps/api/event";
import { Window } from "@tauri-apps/api/window";
import { useCallback, useEffect, useId, useMemo } from "react";
import { useDispatch, useStore } from "react-redux";

import { select } from "@/layout/selectors";
import { createMosaicWindow, moveMosaicTab, type StoreState } from "@/layout/slice";
import { usePlacer } from "@/layout/usePlacer";
import { Runtime } from "@/runtime";

const useWindowsContains = (): ((cursor: xy.XY) => boolean) => {
  const store = useStore<Drift.StoreState>();
  return useCallback(
    (cursor) => {
      const windows = Drift.selectWindows(store.getState());
      const boxes = windows
        .filter((w) => w.stage === "created" && w.reserved)
        .map((w) => box.construct(w.position ?? xy.ZERO, w.size));
      return boxes.some((b) => box.contains(b, cursor));
    },
    [store],
  );
};

const useDropOutsideMacOS = ({
  onDrop,
  canDrop,
  key,
  type,
}: Haul.UseDropOutsideProps) => {
  const ctx = Haul.useContext();
  if (ctx == null) return;
  const { drop } = ctx;
  const dragging = Haul.useDraggingRef();
  const key_ = key ?? useId();
  const target: Haul.Item = useMemo(() => ({ key: key_, type }), [key_, type]);
  const windowsContain = useWindowsContains();
  const store = useStore<StoreState & Drift.StoreState>();
  const handleError = Status.useErrorHandler();
  useAsyncEffect(async () => {
    if (Runtime.ENGINE !== "tauri") return;
    return listen("mouse_up", ({ payload: [x, y] }: { payload: [number, number] }) => {
      handleError(async () => {
        if (dragging.current.items.length === 0 || !canDrop(dragging.current)) return;
        const state = store.getState();
        const layout = select(state, dragging.current.items[0].key as string);
        if (layout?.windowKey == null) return;
        const winLabel = Drift.selectWindowLabel(state, layout.windowKey);
        if (winLabel == null || winLabel !== Drift.MAIN_WINDOW) return;
        const win = await Window.getByLabel(winLabel);
        if (win == null) return;
        const cursor = xy.construct(x, y);
        if (windowsContain(cursor)) return;
        const dropped = onDrop(dragging.current, cursor);
        drop({ target, dropped });
      }, "Failed to drop outside");
    });
  }, [target]);
};

interface UseDropOutsideProps extends Omit<Haul.UseDropProps, "onDrop"> {
  onDrop: (props: Haul.OnDropProps, cursor: xy.XY) => Haul.Item[];
}

const useDropOutsideWindows = ({ type, key, ...rest }: UseDropOutsideProps): void => {
  const ctx = Haul.useContext();
  if (ctx == null) return;
  const { bind } = ctx;
  const key_ = key ?? useId();
  const target: Haul.Item = useMemo(() => ({ key: key_, type }), [key_, type]);
  const propsRef = useSyncedRef<UseDropOutsideProps>({ ...rest, type, key: key_ });
  const windowsContain = useWindowsContains();
  useEffect(() => {
    const release = bind((state, cursor) => {
      const { canDrop, onDrop } = propsRef.current;
      if (!canDrop(state) || windowsContain(cursor)) return null;
      const dropped = onDrop({ ...state }, cursor);
      return { target, dropped };
    });
    return () => {
      release();
    };
  }, []);
};

const canDrop: Haul.CanDrop = ({ items }) =>
  items.length === 1 && items[0].type === Mosaic.HAUL_DROP_TYPE;

const useBase =
  runtime.getOS() === "macOS" ? useDropOutsideMacOS : useDropOutsideWindows;

export const useDropOutside = (): void => {
  const place = usePlacer();
  const dispatch = useDispatch();
  const handleDrop = useCallback(
    ({ items: [item] }: Haul.OnDropProps, cursor?: xy.XY) => {
      if (item == null) return [];
      const { key } = place(
        createMosaicWindow({
          position: cursor ? xy.translate(cursor, { x: -80, y: -45 }) : undefined,
        }),
      );
      dispatch(
        moveMosaicTab({
          windowKey: key,
          key: 1,
          tabKey: item.key as string,
          loc: "center",
        }),
      );
      return [item];
    },
    [place],
  );
  const dropProps = {
    type: "Palette",
    canDrop,
    onDrop: handleDrop,
  };
  useBase(dropProps);
};
