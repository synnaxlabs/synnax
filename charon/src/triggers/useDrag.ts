// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, xy } from "@synnaxlabs/x";
import { type RefObject, useCallback, useRef } from "react";

import { use, type UseEvent } from "@/triggers/hooks";
import { type Stage, type Trigger } from "@/triggers/triggers";

export interface DragEvent {
  stage: Stage;
  box: box.Box;
  cursor: xy.XY;
  triggers: Trigger[];
}

export type DragCallback = (props: DragEvent) => void;

export interface UseDragProps {
  bound: RefObject<HTMLElement | null>;
  triggers?: Trigger[];
  onDrag: DragCallback;
  loose?: boolean;
}

export const useDrag = ({
  onDrag,
  triggers = [["MouseLeft"], ["MouseRight"]],
  bound,
  loose = false,
}: UseDragProps): void => {
  const triggerRef = useRef<UseEvent | null>(null);
  const startLoc = useRef<xy.XY>(xy.ZERO);
  const onMove = useCallback(
    (e: xy.Client & { buttons: number }) => {
      const cursor = xy.construct(e);
      if (triggerRef.current === null) return;
      const { triggers } = triggerRef.current;
      onDrag({
        box: box.construct(startLoc.current, cursor),
        cursor,
        triggers,
        stage: "during",
      });
    },
    [onDrag],
  );
  const handleTrigger = useCallback(
    (event: UseEvent): void => {
      const { stage, cursor } = event;
      if (stage === "start") {
        onDrag({ box: box.construct(cursor), ...event });
        window.addEventListener("mousemove", onMove);
        triggerRef.current = event;
        startLoc.current = cursor;
      } else if (stage === "end" && triggerRef.current != null) {
        onDrag({ box: box.construct(startLoc.current, cursor), ...event });
        window.removeEventListener("mousemove", onMove);
        triggerRef.current = null;
        startLoc.current = xy.ZERO;
      }
    },
    [onDrag],
  );
  use({ triggers, callback: handleTrigger, region: bound, loose });
};
