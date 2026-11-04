// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/device/ConnectFrame.css";

import { Modal } from "@synnaxlabs/lyra/modal";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";

/**
 * The frame of each step of the device connect flow. Every step has the same width, so
 * the modal keeps its size as the person moves between steps.
 */
export const ConnectFrame = ({
  className,
  ...rest
}: Modal.FrameProps): ReactElement => (
  <Modal.Frame
    className={CSS.cls(CSS.B("device-connect-frame"), className)}
    {...rest}
  />
);
