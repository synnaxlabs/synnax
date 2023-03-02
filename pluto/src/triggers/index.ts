// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  matchTriggers,
  TriggersProvider,
  useTrigger,
  useTriggerHeld,
} from "./TriggersContext";
import { useTriggerDrag } from "./useTriggerDrag";
export type { Trigger, Modifier, Key, Stage } from "./types";
export type { TriggerEvent, TriggerCallback } from "./TriggersContext";
export type { TriggerDragEvent, TriggerDragCallback } from "./useTriggerDrag";
export { mouseButtonKey } from "./mouse";

export const Triggers = {
  Provider: TriggersProvider,
  use: useTrigger,
  useHeld: useTriggerHeld,
  useDrag: useTriggerDrag,
  match: matchTriggers,
};
