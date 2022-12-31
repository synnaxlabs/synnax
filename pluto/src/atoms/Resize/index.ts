// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Resize as CoreResize } from "./Resize";
import { ResizeMultiple, useResizeMultiple } from "./ResizeMultiple";

export type { UseResizeMultipleProps } from "./ResizeMultiple";

type CoreResizeType = typeof CoreResize;

interface ResizeType extends CoreResizeType {
  Multiple: typeof ResizeMultiple;
  useMultiple: typeof useResizeMultiple;
}

/**
 * A panel that can be resized in one direction by dragging its handle.
 *
 * @param props - The component props. All unused props will be passed to the div
 * containing the content.
 * @param props.location - The the location of the panel. The drag handle will be on the opposite side.
 * @param props.minSize - The smallest size the panel can be resized to.
 * @param props.maxSize - The largest size the panel can be resized to.
 * @param props.onResize - A callback executed when the panel is resized.
 */
export const Resize = CoreResize as ResizeType;

/**
 * A set of panels that can be resized within their container. Resize.Multiple must be
 * used in conjunction with {@link Resize.useMultiple}.
 *
 * @param props - The component props. All unused props will be passed to the div
 * containing the div containing the panels. Generally these props should not be provided
 * directly, and you should instead spread the props returned from {@link Resize.useMultiple}.
 * The only exceptions to this are stylistic props (e.g. className, style, etc.) as well
 * as the `children` prop.
 */
Resize.Multiple = ResizeMultiple;

/**
 * A hook that implements the control logic for {@link Resize.Multiple}. This hook
 * should be used in conjunction with {@link Resize.Multiple}.
 *
 * @param props - The a component props.
 * @param props.count - The number of panels to be resized. This should be the same as the
 * number of children passed to {@link Resize.Multiple}.
 * @param props.onResize - A callback executed when the panels are resized.
 * @param props.initialSizes - The initial sizes of the panels. This should be an array of
 * numbers, where each number is the initial size of the corresponding panel. If this array
 * is not provided (or is shorter than the expected length), the (remaining) panels will
 * be evenly distributed across the container.
 * @param props.direction - The direction in which the panels should be draggable. This should
 * be Default: "horizontal"
 * @param props.minSize - The smallest size the panels can be resized to. Defaults to 100.
 *
 * @returns The props that should be passed to {@link Resize.Multiple} along with a few
 * utility functions. setSize can be used to manually set the size of a particular panel.
 */
Resize.useMultiple = useResizeMultiple;
