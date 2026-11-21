// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type PropsWithChildren,
  type ReactElement,
  useLayoutEffect,
  useState,
} from "react";
import { createPortal } from "react-dom";

import { createDetachedElement } from "@/portal/detached";

/**
 * Dispatch this event on the document after a modal dialog opens or closes, so each
 * {@link Above} moves its content.
 */
export const MODAL_CHANGE_EVENT = "lyra-modal-change";

const topModal = (): HTMLDialogElement | null => {
  const modals = document.querySelectorAll<HTMLDialogElement>("dialog:modal");
  return modals[modals.length - 1] ?? null;
};

export interface AboveProps extends PropsWithChildren {
  /** The element that hosts the content while no modal dialog is open. */
  element: HTMLElement;
}

/**
 * Shows its children above every open modal dialog, such as a notification. The
 * content moves into the last modal opened, where it stays usable, and back to element
 * when the modals close. It keeps its state as it moves. A modal opened later must come
 * later in the document, as one opened from another does.
 */
export const Above = ({ element, children }: AboveProps): ReactElement => {
  const [host] = useState(createDetachedElement);
  useLayoutEffect(() => {
    const attach = (): void => {
      (topModal() ?? element).appendChild(host);
    };
    attach();
    document.addEventListener(MODAL_CHANGE_EVENT, attach);
    return () => {
      document.removeEventListener(MODAL_CHANGE_EVENT, attach);
      host.remove();
    };
  }, [element, host]);
  return createPortal(children, host);
};
