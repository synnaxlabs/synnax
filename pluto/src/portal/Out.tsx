// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useLayoutEffect, useRef, useState } from "react";

import { type Node } from "@/portal/Node";
import { useContext } from "@/portal/Provider";

export interface OutProps {
  /**
   * itemKey addresses the {@link In} content to host. While null or not yet
   * registered, the Out renders an empty placeholder and attaches the content
   * as soon as it appears.
   */
  itemKey?: string | null;
}

/**
 * Out hosts the content of the {@link In} registered under itemKey at its own
 * position in the DOM. When several Outs address the same key, the last one
 * mounted hosts the content.
 */
export const Out = ({ itemKey }: OutProps): ReactElement => {
  const registry = useContext("Portal.Out");
  const [node, setNode] = useState<Node | undefined>(undefined);
  // Resolution happens in a layout effect rather than useSyncExternalStore so
  // an In registering in the same commit attaches its content before the
  // browser paints, regardless of which side mounts first.
  useLayoutEffect(() => {
    if (itemKey == null) {
      setNode(undefined);
      return;
    }
    setNode(registry.get(itemKey));
    return registry.subscribe(() => setNode(registry.get(itemKey)));
  }, [registry, itemKey]);
  const stub = useRef<HTMLDivElement>(null);
  const hosted = useRef<Node | undefined>(undefined);
  useLayoutEffect(() => {
    if (hosted.current !== node) {
      hosted.current?.unmount(stub.current);
      hosted.current = node;
    }
    const placeholder = stub.current;
    if (node == null || placeholder == null) return;
    const parent = placeholder.parentNode;
    if (parent == null) return;
    node.mount(parent, placeholder);
  }, [node]);
  useLayoutEffect(
    () => () => {
      // Release hosted.current, not a closed-over node: the resolved node may
      // have been swapped since mount.
      if (stub.current != null) hosted.current?.unmount(stub.current);
    },
    [],
  );
  return <div ref={stub} />;
};
