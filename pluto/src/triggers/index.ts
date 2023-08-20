// Copyright 2023 Synnax Labs, Inc.
//
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useTrigger, useTriggerHeld, useTriggerHeldRef } from "@/triggers/hooks";
import {
  keyboardTriggerKey as keyboardKey,
  match,
  filter,
  purge,
  mouseButtonTriggerKey as mouseKey,
  eventTriggerKey as eventKey,
  diff,
  determineTriggerMode,
  reduceTriggerConfig,
} from "@/triggers/triggers";
import { TriggersProvider } from "@/triggers/TriggersContext";
import { TriggerStatus } from "@/triggers/TriggerStatus";
import { useTriggerDrag } from "@/triggers/useTriggerDrag";
export type { TriggersProviderProps } from "@/triggers/TriggersContext";
export type {
  Trigger,
  TriggerKey,
  Stage,
  TriggerEvent,
  TriggerCallback,
} from "@/triggers/triggers";
export type {
  TriggerDragEvent,
  TriggerDragCallback,
} from "@/triggers/useTriggerDrag";
export type { UseTriggerEvent } from "@/triggers/hooks";

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
  determineMode: determineTriggerMode,
  reduceConfig: reduceTriggerConfig,
};
