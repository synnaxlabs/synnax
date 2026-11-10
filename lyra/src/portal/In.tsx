// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor } from "@synnaxlabs/x";
import {
  type ReactElement,
  type ReactNode,
  useLayoutEffect,
  useState,
  useSyncExternalStore,
} from "react";
import { createPortal } from "react-dom";

import { context } from "@/context";
import { useContext } from "@/portal/Context";

export interface InProps {
  /**
   * itemKey identifies the content so an {@link Out} with the same key can
   * host it. Must be unique within the enclosing Context.
   */
  itemKey: string;
  /** children renders the content registered under itemKey. */
  children: ReactNode;
}

// display: contents keeps the element out of layout entirely, so children size and
// position against the hosting Out's element as if they were its direct children.
const createDetachedElement = (): HTMLElement => {
  const el = document.createElement("div");
  el.style.display = "contents";
  return el;
};

interface Attachment {
  subscribe: (listener: () => void) => destructor.Destructor;
  get: () => boolean;
}

// Content outside an In renders in place, so it is always on the page.
const IN_PLACE: Attachment = { subscribe: () => () => {}, get: () => true };

const [AttachmentContext, useAttachment] = context.create<Attachment>({
  defaultValue: IN_PLACE,
  displayName: "Portal.Attachment",
});

/**
 * Returns true while an {@link Out} hosts the content of the enclosing {@link In}, and
 * false while the content waits detached. Returns true outside an In. Detached content
 * cannot take focus, so focus an element once this turns true.
 */
export const useAttached = (): boolean => {
  const { subscribe, get } = useAttachment();
  return useSyncExternalStore(subscribe, get);
};

/**
 * In renders children into a detached element registered under itemKey in the enclosing
 * {@link Context}. The content stays mounted at the In's position in the React tree for
 * the In's whole lifetime, while {@link Out} parts with the same key host it in the
 * DOM. Because the element is moved rather than recreated when its host changes, the
 * content keeps its state (DOM, WebGL contexts) across moves.
 */
export const In = ({ itemKey, children }: InProps): ReactElement => {
  const registry = useContext("Portal.In");
  const [el] = useState(createDetachedElement);
  useLayoutEffect(() => {
    registry.register(itemKey, el);
    return () => {
      registry.unregister(itemKey);
      el.remove();
    };
  }, [registry, itemKey, el]);
  const attachment: Attachment = {
    subscribe: (listener) => registry.subscribeHosted(listener, itemKey),
    get: () => el.isConnected,
  };
  return createPortal(
    <AttachmentContext value={attachment}>{children}</AttachmentContext>,
    el,
  );
};
