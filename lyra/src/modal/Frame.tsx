// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/modal/Frame.css";

import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Dialog } from "@/dialog";

export interface FrameProps extends Dialog.DialogProps {}

/**
 * Frame is the surface of a modal: a {@link Dialog.Dialog} sized to its content and
 * capped at the viewport, holding a {@link Header}, a {@link Body}, and a
 * {@link Footer}.
 */
export const Frame = ({ className, ...rest }: FrameProps): ReactElement => (
  <Dialog.Dialog className={CSS.cls(CSS.B("modal"), className)} {...rest} />
);
