// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/groupBox/groupBox.css";

import { type dimensions } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";

export interface PrimitiveProps {
  dimensions: dimensions.Dimensions;
  className?: string;
}

export const Primitive = ({ dimensions, className }: PrimitiveProps): ReactElement => (
  <div
    className={CSS.cls(className, CSS.B("group-box"))}
    style={{ width: dimensions.width, height: dimensions.height }}
  />
);
