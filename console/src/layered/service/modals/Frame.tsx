// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dialog } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/css";

export interface FrameProps extends Dialog.DialogProps {}

export const Frame = ({ className, ...rest }: FrameProps): ReactElement => (
  <Dialog.Dialog className={CSS(CSS.B("modal"), className)} {...rest} />
);
