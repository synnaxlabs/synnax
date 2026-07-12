// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, type ReactNode, useLayoutEffect, useState } from "react";
import { createPortal } from "react-dom";

import { useSyncedRef } from "@/hooks";
import { useContext } from "@/portal/Context";
import { Node } from "@/portal/Node";

export interface InProps {
  /**
   * itemKey identifies the content so an {@link Out} with the same key can
   * host it. Must be unique within the enclosing Context.
   */
  itemKey: string;
  /**
   * attrs are set as attributes on the content's element when it is created.
   * Changing them later has no effect.
   */
  attrs?: Record<string, string>;
  /**
   * onClick is invoked with itemKey when the content is clicked. Clicks inside
   * portaled content bubble through the React tree of the In, not the DOM tree
   * of the hosting Out, so the handler is bound natively on the element itself.
   */
  onClick?: (key: string) => void;
  /** children renders the content registered under itemKey. */
  children: ReactNode;
}

/**
 * In renders children into a detached element registered under itemKey in the
 * enclosing {@link Context}. The content stays mounted at the In's position
 * in the React tree for the In's whole lifetime, while {@link Out} parts with
 * the same key host it in the DOM. Because the element is moved rather than
 * recreated when its host changes, the content keeps its state (DOM, WebGL
 * contexts) across moves.
 */
export const In = ({ itemKey, attrs, onClick, children }: InProps): ReactElement => {
  const registry = useContext("Portal.In");
  const [node] = useState(() => new Node(attrs));
  const onClickRef = useSyncedRef(onClick);
  const keyRef = useSyncedRef(itemKey);
  useLayoutEffect(() => {
    const handleClick = (): void => onClickRef.current?.(keyRef.current);
    node.el.addEventListener("click", handleClick);
    return () => node.el.removeEventListener("click", handleClick);
  }, [node]);
  useLayoutEffect(() => {
    registry.register(itemKey, node);
    return () => {
      registry.unregister(itemKey);
      node.unmount(null);
    };
  }, [registry, itemKey, node]);
  return createPortal(children, node.el);
};
