// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { type ReactNode, useCallback } from "react";

import { CSS } from "@/css";
import { List } from "@/list";
import { useRegistryContext, useVisibleCount } from "@/select/registry";
import { useClosed } from "@/select/scope";

/** Props for {@link Items}. */
export interface ItemsProps<
  K extends record.Key = record.Key,
> extends List.ItemsProps<K> {}

/**
 * Renders the frame's data as options, in its place among the fixed {@link Item}s. It
 * renders nothing while the dialog is closed, and shows `emptyContent` only while no
 * fixed item is visible.
 *
 * @throws if no `Select.List` of the same frame encloses it.
 */
export const Items = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K>,
>({
  children,
  emptyContent,
}: ItemsProps<K>): ReactNode => {
  const closed = useClosed();
  const registry = useRegistryContext("Select.Items");
  const fixed = useVisibleCount();
  const markerRef = useCallback(
    (element: HTMLElement | null) => {
      registry.setBlock(element);
      return () => registry.setBlock(null);
    },
    [registry],
  );
  if (closed) return null;
  return (
    <>
      <span ref={markerRef} className={CSS.BE("select", "block")} hidden />
      <List.Items<K, E> emptyContent={fixed > 0 ? null : emptyContent}>
        {children}
      </List.Items>
    </>
  );
};
