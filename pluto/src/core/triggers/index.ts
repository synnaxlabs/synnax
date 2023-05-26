// Copyright 2023 Synnax Labs, Inc.
//
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useTrigger, useTriggerHeld, useTriggerHeldRef } from "./hooks";
import {
  keyboardToKey as keyboardKey,
  match,
  filter,
  purge,
  mouseButtonToKey as mouseKey,
  parseEventKey as eventKey,
  diff,
} from "./triggers";
import { TriggersProvider } from "./TriggersContext";
import { TriggerStatus } from "./TriggerStatus";
import { useTriggerDrag } from "./useTriggerDrag";
export type { TriggersProviderProps } from "./TriggersContext";
export type { Trigger, Key, Stage, TriggerEvent, TriggerCallback } from "./triggers";
export type { TriggerDragEvent, TriggerDragCallback } from "./useTriggerDrag";

export const Triggers = {
  Provider: TriggersProvider,
  Status: TriggerStatus,
  match,
  filter,
  purge,
  diff,
  use: useTrigger,
  useHeld: useTriggerHeld,
  useHeldRef: useTriggerHeldRef,
  useDrag: useTriggerDrag,
  mouseKey,
  keyboardKey,
  eventKey,
};
