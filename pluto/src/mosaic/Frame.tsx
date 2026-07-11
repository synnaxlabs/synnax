// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/mosaic/Mosaic.css";

import { type location } from "@synnaxlabs/x";
import { type DragEvent, type ReactElement, useMemo } from "react";

import { context } from "@/context";
import { CSS } from "@/css";
import { Flex } from "@/flex";

/** Passed to a Frame's onDrop handler when a tab lands on a {@link Leaf}. */
export interface OnDropProps {
  /** The key of the leaf the tab was dropped on. */
  leafKey: string;
  /** The key of the dropped tab. */
  tabKey: string;
  /** Where the tab landed relative to the leaf: an edge splits, center inserts. */
  location: location.Location;
  /** The insertion index within the leaf's tab strip, when dropped on the strip. */
  index?: number;
}

/** Passed to a Frame's onCreate handler when creation items land on a {@link Leaf}. */
export interface OnCreateProps {
  /** The key of the leaf the items were dropped on. */
  leafKey: string;
  /** Where the items landed relative to the leaf: an edge splits, center inserts. */
  location: location.Location;
  /** The keys of the dropped creation items, e.g. ontology ID strings. */
  tabKeys: string[];
  /** The insertion index within the leaf's tab strip, when dropped on the strip. */
  index?: number;
}

/** Passed to a Frame's onFileDrop handler when OS files land on a {@link Leaf}. */
export interface OnFileDropProps {
  /** The key of the leaf the files were dropped on. */
  leafKey: string;
  /** Where the files landed relative to the leaf: an edge splits, center inserts. */
  location: location.Location;
  /** The native drag event carrying the dropped files. */
  event: DragEvent;
}

export interface ContextValue {
  /** onDrop is the Frame-level tab drop handler invoked by {@link Leaf} parts. */
  onDrop?: (props: OnDropProps) => void;
  /** onCreate is the Frame-level creation drop handler invoked by {@link Leaf} parts. */
  onCreate?: (props: OnCreateProps) => void;
  /**
   * onFileDrop is the Frame-level OS file drop handler invoked by {@link Leaf}
   * parts. Its presence enables file drops on every leaf in the frame.
   */
  onFileDrop?: (props: OnFileDropProps) => void;
  /** onResize is the Frame-level resize handler invoked by {@link Split} parts. */
  onResize?: (splitKey: string, size: number) => void;
}

const [Context, useContext] = context.create<ContextValue>({
  displayName: "Mosaic.Context",
  providerName: "Mosaic.Frame",
});

export { useContext };

export interface FrameProps extends Omit<Flex.BoxProps, "onDrop" | "onResize"> {
  /** onDrop is called when a hauled tab lands on one of the frame's leaves. */
  onDrop?: (props: OnDropProps) => void;
  /** onCreate is called when hauled creation items land on one of the frame's leaves. */
  onCreate?: (props: OnCreateProps) => void;
  /** onFileDrop enables OS file drops on the frame's leaves and handles them. */
  onFileDrop?: (props: OnFileDropProps) => void;
  /** onResize is called when a {@link Split} handle drag ends with the new ratio. */
  onResize?: (splitKey: string, size: number) => void;
}

/**
 * Frame is the root of a composed mosaic. It distributes drop, create, file drop,
 * and resize handlers to the Leaf and Split parts rendered within it. Consumers
 * render their own tree of Split and Leaf parts as children.
 */
export const Frame = ({
  onDrop,
  onCreate,
  onFileDrop,
  onResize,
  className,
  children,
  empty = true,
  ...rest
}: FrameProps): ReactElement => {
  const ctx = useMemo<ContextValue>(
    () => ({ onDrop, onCreate, onFileDrop, onResize }),
    [onDrop, onCreate, onFileDrop, onResize],
  );
  return (
    <Context value={ctx}>
      <Flex.Box empty={empty} className={CSS(CSS.B("mosaic"), className)} {...rest}>
        {children}
      </Flex.Box>
    </Context>
  );
};
