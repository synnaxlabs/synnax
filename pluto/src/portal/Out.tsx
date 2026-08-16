// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type ComponentPropsWithoutRef,
  type ReactElement,
  useLayoutEffect,
  useRef,
} from "react";

import { useSyncedRef } from "@/hooks";
import { useContext } from "@/portal/Context";

// React synthetic events on hosted content bubble through the React tree of its
// In, not the DOM tree of the host, so handlers passed here are bound natively
// on the host element to catch interaction with whatever it currently hosts.
// The Capture variant binds in the capture phase, so it fires before any handler
// inside the hosted content and can act ahead of it (e.g. focusing this host's
// tab before a same-click navigation the content triggers).
const DELEGATED_EVENTS = {
  onClick: { name: "click", capture: false },
  onClickCapture: { name: "click", capture: true },
  onDoubleClick: { name: "dblclick", capture: false },
  onContextMenu: { name: "contextmenu", capture: false },
  onPointerDown: { name: "pointerdown", capture: false },
  onPointerUp: { name: "pointerup", capture: false },
  onMouseDown: { name: "mousedown", capture: false },
  onMouseUp: { name: "mouseup", capture: false },
  onMouseEnter: { name: "mouseenter", capture: false },
  onMouseLeave: { name: "mouseleave", capture: false },
  onKeyDown: { name: "keydown", capture: false },
  onKeyUp: { name: "keyup", capture: false },
} as const satisfies Record<
  string,
  { name: keyof HTMLElementEventMap; capture: boolean }
>;

type DelegatedEventMap = typeof DELEGATED_EVENTS;

/**
 * HostHandlers mirrors the matching React handler props on a div, but each
 * receives the native event since it fires from a natively bound listener.
 */
export type HostHandlers = {
  [K in keyof DelegatedEventMap]?: (
    ev: HTMLElementEventMap[DelegatedEventMap[K]["name"]],
  ) => void;
};

export interface OutProps
  extends
    Omit<ComponentPropsWithoutRef<"div">, "children" | keyof HostHandlers>,
    HostHandlers {
  /**
   * itemKey addresses the {@link In} content to host. While null or not yet
   * registered, the Out renders an empty host and attaches the content as
   * soon as it appears.
   */
  itemKey?: string | null;
}

/**
 * Out hosts the content of the {@link In} registered under itemKey at its own
 * position in the DOM. When several Outs address the same key, the last one
 * mounted hosts the content. Handler props ({@link HostHandlers}) fire for
 * interaction with the hosted content; remaining props are forwarded to the
 * host div, whose children the content lays out as.
 */
export const Out = ({
  itemKey,
  onClick,
  onClickCapture,
  onDoubleClick,
  onContextMenu,
  onPointerDown,
  onPointerUp,
  onMouseDown,
  onMouseUp,
  onMouseEnter,
  onMouseLeave,
  onKeyDown,
  onKeyUp,
  ...rest
}: OutProps): ReactElement => {
  const registry = useContext("Portal.Out");
  const host = useRef<HTMLDivElement>(null);
  const handlers = useSyncedRef<HostHandlers>({
    onClick,
    onClickCapture,
    onDoubleClick,
    onContextMenu,
    onPointerDown,
    onPointerUp,
    onMouseDown,
    onMouseUp,
    onMouseEnter,
    onMouseLeave,
    onKeyDown,
    onKeyUp,
  });
  useLayoutEffect(() => {
    const hostEl = host.current;
    if (hostEl == null) return;
    const bound = Object.entries(DELEGATED_EVENTS).map(
      ([reactName, { name, capture }]) => {
        const listener = (ev: Event): void =>
          (
            handlers.current[reactName as keyof HostHandlers] as
              ((e: Event) => void) | undefined
          )?.(ev);
        hostEl.addEventListener(name, listener, capture);
        return () => hostEl.removeEventListener(name, listener, capture);
      },
    );
    return () => bound.forEach((unbind) => unbind());
  }, []);
  // Attachment is fully imperative: the hosted element never appears in JSX,
  // so routing it through state would only add a second pre-paint render on
  // mount and a re-render per registry change. A layout effect (rather than
  // useEffect) attaches content before the browser paints regardless of which
  // side of the In/Out pair mounts first within a commit.
  useLayoutEffect(() => {
    const hostEl = host.current;
    if (itemKey == null || hostEl == null) return;
    let hosted: HTMLElement | undefined;
    const attach = (): void => {
      const el = registry.get(itemKey);
      if (el === hosted) return;
      if (hosted?.parentNode === hostEl) hosted.remove();
      hosted = el;
      // appendChild moves an element that is already in the document, so
      // claiming the content automatically releases its previous host.
      if (el != null) hostEl.appendChild(el);
    };
    attach();
    const unsubscribe = registry.subscribe(attach, itemKey);
    return () => {
      unsubscribe();
      if (hosted?.parentNode === hostEl) hosted.remove();
    };
  }, [registry, itemKey]);
  return <div ref={host} {...rest} />;
};
