// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/tooltip/Frame.css";

import { box, location } from "@synnaxlabs/x";
import {
  type ReactElement,
  type ReactNode,
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
} from "react";
import { createPortal } from "react-dom";

import { CSS } from "@/css";
import { useCombinedRefs, useResize, useSyncedRef, useWindowResize } from "@/hooks";
import { position } from "@/position";
import { getRootElement } from "@/util/rootElement";

export interface FrameProps {
  /** The element the frame sits beside. */
  anchor: HTMLElement;
  /** The preferred location relative to the anchor. Chosen by position when unset. */
  location?: position.Location;
  id?: string;
  className?: string;
  /** Called when the anchor leaves the document or collapses to zero area. */
  onAnchorLost?: () => void;
  children: ReactNode;
}

const PREFERENCES: position.LocationPreference[] = [
  { targetCorner: location.TOP_CENTER, dialogCorner: location.BOTTOM_CENTER },
  { targetCorner: location.BOTTOM_CENTER, dialogCorner: location.TOP_CENTER },
  { targetCorner: location.CENTER_RIGHT, dialogCorner: location.CENTER_LEFT },
  { targetCorner: location.CENTER_LEFT, dialogCorner: location.CENTER_RIGHT },
];
const OFFSET = 6;

/** The positioned tooltip box, following its anchor and content through resizes. */
export const Frame = ({
  anchor,
  location: locationProp,
  id,
  className,
  onAnchorLost,
  children,
}: FrameProps): ReactElement => {
  const ref = useRef<HTMLDivElement | null>(null);
  const prevPlacementRef = useRef<position.Preference | null>(null);
  const locationRef = useSyncedRef(locationProp);
  const onAnchorLostRef = useSyncedRef(onAnchorLost);
  const reposition = useCallback((): void => {
    const el = ref.current;
    if (el == null) return;
    const target = box.construct(anchor);
    if (!anchor.isConnected || box.areaIsZero(target))
      return onAnchorLostRef.current?.();
    let prefer = PREFERENCES;
    if (prevPlacementRef.current != null)
      prefer = [prevPlacementRef.current, ...PREFERENCES];
    const { adjustedDialog, ...placement } = position.position({
      target,
      // offset* ignores the entrance animation's scale, unlike getBoundingClientRect.
      dialog: box.construct(0, 0, el.offsetWidth, el.offsetHeight),
      container: box.construct(0, 0, window.innerWidth, window.innerHeight),
      initial: locationRef.current,
      prefer,
      offset: OFFSET,
    });
    prevPlacementRef.current = placement;
    const rounded = box.round(adjustedDialog);
    el.style.left = CSS.px(box.left(rounded));
    el.style.top = CSS.px(box.top(rounded));
    el.style.transformOrigin = `${placement.dialogCorner.x} ${placement.dialogCorner.y}`;
  }, [anchor]);
  useLayoutEffect(reposition);
  const observeAnchor = useResize(reposition);
  useEffect(() => observeAnchor(anchor), [anchor, observeAnchor]);
  const resizeRef = useResize(reposition);
  useWindowResize(reposition);
  const combinedRef = useCombinedRefs(ref, resizeRef);
  return createPortal(
    <div
      id={id}
      role="tooltip"
      ref={combinedRef}
      className={CSS.cls(CSS.B("tooltip"), className)}
    >
      {children}
    </div>,
    getRootElement(),
  );
};
