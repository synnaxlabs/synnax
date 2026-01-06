// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  setWindowFullscreen,
  setWindowMaximized,
  setWindowMinimized,
} from "@synnaxlabs/drift";
import { useSelectWindow } from "@synnaxlabs/drift/react";
import { OS } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { useRemover } from "@/layout/useRemover";
import { Runtime } from "@/runtime";

export interface ControlsProps extends OS.ControlsProps {}

export const Controls = (props: ControlsProps): ReactElement | null => {
  const os = OS.use();
  const window = useSelectWindow();
  const dispatch = useDispatch();
  const remove = useRemover(window?.key ?? "");
  const maximizedDisabled = window?.resizable === false;
  const disabled: OS.ControlsAction[] = [];
  if (window?.focus === false && os === "macOS")
    disabled.push("close", "minimize", "maximize");
  else if (maximizedDisabled) disabled.push("maximize");
  const handleClose = useCallback(() => {
    remove();
  }, [remove]);
  const handleFullscreen = useCallback(() => {
    dispatch(setWindowFullscreen({}));
  }, [dispatch]);
  const handleMaximize = useCallback(() => {
    dispatch(setWindowMaximized({}));
  }, [dispatch]);
  const handleMinimize = useCallback(() => {
    dispatch(setWindowMinimized({ value: true }));
  }, [dispatch]);
  if (Runtime.ENGINE !== "tauri") return null;
  return window?.fullscreen === true ? null : (
    <OS.Controls
      disabled={disabled}
      focused={window?.focus}
      onClose={handleClose}
      onFullscreen={handleFullscreen}
      onMaximize={handleMaximize}
      onMinimize={handleMinimize}
      {...props}
    />
  );
};
