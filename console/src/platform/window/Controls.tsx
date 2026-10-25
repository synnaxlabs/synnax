// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/window/Controls.css";

import { Drift, setWindowMaximized, setWindowMinimized } from "@synnaxlabs/drift";
import { useSelectWindow } from "@synnaxlabs/drift/react";
import { OS } from "@synnaxlabs/pluto";
import { type runtime } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { Buttons, type ButtonsProps } from "@/platform/window/Buttons";
import { Session } from "@/session";

export interface ControlsProps extends ButtonsProps {
  /** The OS to draw for, in place of the one the Console runs on. */
  forceOS?: runtime.OS;
  /** Draws nothing unless the OS is this one. */
  visibleIfOS?: runtime.OS;
}

/**
 * The window's controls. On macOS the window draws its native controls, so this holds
 * their place in the layout.
 */
export const Controls = ({
  visibleIfOS,
  forceOS,
  ...rest
}: ControlsProps): ReactElement | null => {
  const os = OS.use({ force: forceOS });
  const window = useSelectWindow();
  const dispatch = Session.useDispatch();
  const handleClose = () => {
    dispatch(Drift.closeWindow({}));
  };
  const handleMaximize = () => {
    dispatch(setWindowMaximized({}));
  };
  const handleMinimize = () => {
    dispatch(setWindowMinimized({ value: true }));
  };
  if (Session.Runtime.ENGINE !== "tauri" || window?.fullscreen === true) return null;
  if (visibleIfOS != null && visibleIfOS !== os) return null;
  if (os === "macOS")
    return (
      <div className={CSS.BE("window", "native-controls")} data-tauri-drag-region />
    );
  return (
    <Buttons
      disabled={window?.resizable === false ? ["maximize"] : undefined}
      maximized={window?.maximized}
      onClose={handleClose}
      onMaximize={handleMaximize}
      onMinimize={handleMinimize}
      {...rest}
    />
  );
};
