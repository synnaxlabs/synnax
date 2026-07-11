// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useCallback } from "react";

import { useContext } from "@/mosaic/Frame";
import { Resize } from "@/resize";

export interface SplitProps extends Omit<Resize.SplitProps, "onResizeEnd"> {
  /** The key identifying this split, passed to the Frame's onResize handler. */
  splitKey: string;
}

/**
 * Split lays out two panes of a mosaic side by side with a draggable handle
 * between them. When a handle drag ends, the Frame's onResize handler is called
 * with the split's key and the committed ratio. Children are typically Leaf parts
 * or nested Splits.
 */
export const Split = ({ splitKey, ...rest }: SplitProps): ReactElement => {
  const { onResize } = useContext("Mosaic.Split");
  const handleResizeEnd = useCallback(
    (size: number) => onResize?.(splitKey, size),
    [onResize, splitKey],
  );
  return <Resize.Split onResizeEnd={handleResizeEnd} {...rest} />;
};
