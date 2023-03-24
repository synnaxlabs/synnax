// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useTrigger, useTriggerHeld } from "./hooks";
import {
  keyboardToKey as keyboardKey,
  match,
  mouseButtonToKey as mouseKey,
  parseEventKey as eventKey,
} from "./triggers";
import { TriggersProvider } from "./TriggersContext";
import { useTriggerDrag } from "./useTriggerDrag";
export type { Trigger, Key, Stage, TriggerEvent, TriggerCallback } from "./triggers";
export type { TriggerDragEvent, TriggerDragCallback } from "./useTriggerDrag";

export const Triggers = {
  Provider: TriggersProvider,
  match,
  use: useTrigger,
  useHeld: useTriggerHeld,
  useDrag: useTriggerDrag,
  mouseKey,
  keyboardKey,
  eventKey,
};
