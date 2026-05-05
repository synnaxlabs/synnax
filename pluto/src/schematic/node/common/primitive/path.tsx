// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ComponentPropsWithoutRef, type ReactElement } from "react";

export interface PathProps extends ComponentPropsWithoutRef<"path"> {}

export const Path = (props: PathProps): ReactElement => (
  <path vectorEffect="non-scaling-stroke" {...props} />
);

export interface RectProps extends ComponentPropsWithoutRef<"rect"> {}

export const Rect = (props: RectProps): ReactElement => (
  <rect vectorEffect="non-scaling-stroke" {...props} />
);

export interface CircleProps extends ComponentPropsWithoutRef<"circle"> {}

export const Circle = (props: CircleProps): ReactElement => (
  <circle vectorEffect="non-scaling-stroke" {...props} />
);

export interface LineProps extends ComponentPropsWithoutRef<"line"> {}

export const Line = (props: LineProps): ReactElement => (
  <line vectorEffect="non-scaling-stroke" {...props} />
);
