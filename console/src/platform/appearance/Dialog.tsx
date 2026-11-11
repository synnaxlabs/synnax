// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/appearance/Dialog.css";

import { Dialog as Base } from "@synnaxlabs/lyra/dialog";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";

/** Props for {@link Dialog}. */
export interface DialogProps extends Base.DialogProps {}

/** The surface of a {@link Picker}, sized so the swatches keep their spacing. */
export const Dialog = ({ className, ...rest }: DialogProps): ReactElement => (
  <Base.Dialog className={CSS.cls(CSS.B("appearance-dialog"), className)} {...rest} />
);
