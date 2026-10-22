// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Haul } from "@synnaxlabs/lyra/haul";
import { useCallback } from "react";

import { useIsPreview } from "@/platform/task/Form";

/** Builds the new entries of a view from the items dragged onto it. */
export interface Drop<E> extends Pick<Haul.UseDropProps, "type" | "canDrop"> {
  /** @returns the items the drop takes and the entries they build. */
  onDrop: (props: Haul.OnDropProps) => { dropped: Haul.Item[]; added: E[] };
}

export interface UseDropProps<E> {
  /** Undefined for a view that takes no drops. */
  drop?: Drop<E>;
  /** Takes the entries a drop builds. Not called when it builds none. */
  onAdd: (added: E[]) => void;
}

const NEVER: Haul.CanDrop = () => false;

/**
 * Makes an element a drop target that hands the entries a drop builds to onAdd.
 * Returns no handlers when there is no drop or the form is a preview.
 */
export const useDrop = <E>({
  drop,
  onAdd,
}: UseDropProps<E>): Partial<Haul.UseDropReturn> => {
  const isPreview = useIsPreview();
  const handleDrop = useCallback(
    (props: Haul.OnDropProps): Haul.Item[] => {
      const { dropped, added } = drop?.onDrop(props) ?? { dropped: [], added: [] };
      if (added.length > 0) onAdd(added);
      return dropped;
    },
    [drop, onAdd],
  );
  const dropProps = Haul.useDrop({
    type: drop?.type ?? "",
    canDrop: drop?.canDrop ?? NEVER,
    onDrop: handleDrop,
  });
  // A preview hides its own drag sources, but those of another tab still reach it.
  return drop == null || isPreview ? {} : dropProps;
};
