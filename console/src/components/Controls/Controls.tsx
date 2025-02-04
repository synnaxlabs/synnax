// Copyright 2024 Synnax Labs, Inc.
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
import { type ReactElement } from "react";
import { useDispatch } from "react-redux";

import { Layout } from "@/layout";

export interface ControlsProps extends OS.ControlsProps {}

export const Controls = (props: ControlsProps): ReactElement | null => {
  const os = OS.use();
  const window = useSelectWindow();
  const dispatch = useDispatch();
  const remove = Layout.useRemover(window?.key ?? "");
  if (window == null) return null;
  const maximizedDisabled = window.resizable === false;
  const disabled: OS.ControlsAction[] = [];
  if (window.focus === false && os === "macOS")
    disabled.push("close", "minimize", "maximize");
  else if (maximizedDisabled) disabled.push("maximize");

  const handleMinimize = (): void => {
    dispatch(setWindowMinimized({ value: true }));
  };

  const handleMaximize = (): void => {
    dispatch(setWindowMaximized({}));
  };

  const handleFullscreen = (): void => {
    dispatch(setWindowFullscreen({}));
  };

  const handleClose = (): void => {
    remove();
  };

  if (window.fullscreen === true) return null;

  return (
    <OS.Controls
      disabled={disabled}
      focused={window.focus}
      onClose={handleClose}
      onMinimize={handleMinimize}
      onMaximize={handleMaximize}
      onFullscreen={handleFullscreen}
      {...props}
    />
  );
};
