// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Diagram } from "@/vis/diagram";

export interface SlateProps extends Diagram.DiagramProps {}

export const Slate = ({ className, ...props }: SlateProps): ReactElement => (
  <Diagram.Diagram className={CSS(className, CSS.B("slate"))} {...props} />
);
