// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ComponentPropsWithRef, type ReactElement } from "react";

import { CSS } from "@/css";
import { type OrientableProps } from "@/schematic/node/common/primitive/orientable";

export interface DivProps
  extends Omit<ComponentPropsWithRef<"div">, "color" | "onResize">, OrientableProps {}

export const Div = ({ className, ...rest }: DivProps): ReactElement => (
  <div className={CSS.cls(CSS.B("symbol-primitive"), className)} {...rest} />
);
