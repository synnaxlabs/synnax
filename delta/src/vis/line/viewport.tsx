// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  useCallback,
  useDebugValue,
  useEffect,
  useMemo,
  useState,
  useTransition,
} from "react";

import {
  UseViewportReturn as PUseViewportReturn,
  Menu as PMenu,
  Viewport as PViewport,
  UseViewportHandler,
  UseContextMenuReturn,
} from "@synnaxlabs/pluto";
import { Box, DECIMAL_BOX, Deep, XY } from "@synnaxlabs/x";
import { useDispatch } from "react-redux";

import { selectRequiredVis, updateVis, VisualizationStoreState } from "../store";

import { LineVis } from "./LinePlot/core";

import { useMemoSelect } from "@/hooks";
import { LayoutStoreState } from "@/layout";

export interface HoverState {
  cursor: XY;
  box: Box;
}

export interface UseViewportReturn {
  viewportProps: PUseViewportReturn;
  menuProps: UseContextMenuReturn;
  viewport: Box;
  selection: Box | null;
  hover: HoverState | null;
}

export const use = (key: string): UseViewportReturn => {
  const [viewport, setViewport] = useState<Box>(Deep.copy(DECIMAL_BOX));
  const [selection, setSelection] = useState<Box | null>(null);
  const [hover, setHover] = useState<HoverState | null>(null);
  const [, startTransition] = useTransition();

  useDebugValue({ viewport, selection, hover });

  const core = useMemoSelect(
    (state: VisualizationStoreState & LayoutStoreState) =>
      selectRequiredVis<LineVis>(state, key, "line").viewport,
    [key]
  );

  const rdxViewport = useMemo(
    () => new Box(core.pan, core.zoom).reRoot("bottomLeft"),
    [core]
  );

  useEffect(() => {
    setViewport(rdxViewport);
  }, [rdxViewport]);

  const dispatch = useDispatch();

  const updateViewport = useCallback(
    (box: Box) =>
      dispatch(updateVis({ key, viewport: { pan: box.bottomLeft, zoom: box.dims } })),
    [key]
  );

  const menuProps = PMenu.useContextMenu();

  const handleViewport: UseViewportHandler = useCallback(
    (props) => {
      const { box, mode, cursor, stage } = props;
      if (mode === "hover") return setHover({ cursor, box });
      if (mode === "select") {
        setSelection(box);
        return menuProps.open(cursor);
      }
      startTransition(() => {
        setSelection(null);
        setViewport(box);
      });
      if (stage === "end") updateViewport(box);
    },
    [updateViewport]
  );

  const viewportProps = PViewport.use({
    onChange: handleViewport,
    initial: rdxViewport,
  });

  return { viewportProps, menuProps, viewport, selection, hover };
};

export const Viewport = {
  use,
};
