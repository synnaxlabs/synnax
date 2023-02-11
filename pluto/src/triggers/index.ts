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

export const Triggers = {
  Provider: TriggersProvider,
  use: useTrigger,
  useHeld: useTriggerHeld,
  useDrag: useTriggerDrag,
  match: matchTriggers,
};
