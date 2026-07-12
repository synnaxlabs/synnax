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
  useState,
} from "react";

import { useContext } from "@/portal/Context";

export interface OutProps extends Omit<ComponentPropsWithoutRef<"div">, "children"> {
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
 * mounted hosts the content. Remaining props are forwarded to the host div,
 * whose children the content lays out as.
 */
export const Out = ({ itemKey, ...rest }: OutProps): ReactElement => {
  const registry = useContext("Portal.Out");
  const [el, setEl] = useState<HTMLElement | undefined>(undefined);
  // Resolution happens in a layout effect rather than useSyncExternalStore so
  // an In registering in the same commit attaches its content before the
  // browser paints, regardless of which side mounts first.
  useLayoutEffect(() => {
    if (itemKey == null) {
      setEl(undefined);
      return;
    }
    setEl(registry.get(itemKey));
    return registry.subscribe(() => setEl(registry.get(itemKey)));
  }, [registry, itemKey]);
  const host = useRef<HTMLDivElement>(null);
  useLayoutEffect(() => {
    const hostEl = host.current;
    if (el == null || hostEl == null) return;
    // appendChild moves an element that is already in the document, so
    // claiming the content automatically releases its previous host.
    hostEl.appendChild(el);
    return () => {
      if (el.parentNode === hostEl) el.remove();
    };
  }, [el]);
  return <div ref={host} {...rest} />;
};
