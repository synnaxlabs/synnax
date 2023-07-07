// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { RefObject, useCallback, useRef } from "react";

import { Box, ClientXYT, XY } from "@synnaxlabs/x";

import { useTrigger, UseTriggerEvent } from "@/core/triggers/hooks";
import { Stage, Trigger } from "@/core/triggers/triggers";

export interface TriggerDragEvent {
  stage: Stage;
  box: Box;
  cursor: XY;
  triggers: Trigger[];
}

export type TriggerDragCallback = (props: TriggerDragEvent) => void;

export interface UseTriggerDragProps {
  bound: RefObject<HTMLElement>;
  triggers?: Trigger[];
  onDrag: TriggerDragCallback;
}

export const useTriggerDrag = ({
  onDrag,
  triggers = [["MouseLeft"], ["MouseRight"]],
  bound,
}: UseTriggerDragProps): void => {
  const triggerRef = useRef<UseTriggerEvent | null>(null);
  const startLoc = useRef<XY>(XY.ZERO);
  const onMove = useCallback(
    (e: ClientXYT & { buttons: number }) => {
      const cursor = new XY(e);
      if (triggerRef.current === null) return;
      const { triggers } = triggerRef.current;
      onDrag({
        box: new Box(startLoc.current, cursor),
        cursor,
        triggers,
        stage: "during",
      });
    },
    [onDrag]
  );
  const handleTrigger = useCallback(
    (event: UseTriggerEvent): void => {
      const { stage, cursor } = event;
      if (stage === "start") {
        onDrag({ box: new Box(cursor), ...event });
        window.addEventListener("mousemove", onMove);
        triggerRef.current = event;
        startLoc.current = cursor;
      } else if (stage === "end" && triggerRef.current != null) {
        onDrag({ box: new Box(startLoc.current, cursor), ...event });
        window.removeEventListener("mousemove", onMove);
        triggerRef.current = null;
        startLoc.current = XY.ZERO;
      }
    },
    [onDrag]
  );
  useTrigger({ triggers, callback: handleTrigger, region: bound });
};
