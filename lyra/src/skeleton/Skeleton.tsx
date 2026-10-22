// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/skeleton/Skeleton.css";

import {
  type ComponentPropsWithRef,
  type CSSProperties,
  type ReactElement,
} from "react";

import { CSS } from "@/css";

/** Props for {@link Skeleton}. */
export interface SkeletonProps extends Omit<ComponentPropsWithRef<"span">, "children"> {
  /** Defaults to the full width of its parent. */
  width?: CSSProperties["width"];
  /** Defaults to about the height of small text. */
  height?: CSSProperties["height"];
}

/**
 * A pulsing block that holds the place of content still loading, so the layout keeps
 * its shape when the content comes. Screen readers skip it.
 */
export const Skeleton = ({
  width,
  height,
  className,
  style,
  ...rest
}: SkeletonProps): ReactElement => (
  <span
    aria-hidden
    className={CSS.cls(CSS.B("skeleton"), className)}
    style={{ width, height, ...style }}
    {...rest}
  />
);
