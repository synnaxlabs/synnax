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
  const host = useRef<HTMLDivElement>(null);
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
