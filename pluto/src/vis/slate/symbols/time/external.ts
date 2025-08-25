import { Form as IntervalForm } from "@/vis/slate/symbols/time/Form";
import { Interval } from "@/vis/slate/symbols/time/Interval";
import { Schedule } from "@/vis/slate/symbols/time/Schedule";
import { Form as ScheduleForm } from "@/vis/slate/symbols/time/ScheduleForm";
import { type types } from "@/vis/slate/symbols/types";

export const SYMBOLS: Record<string, types.Spec<any>> = {
  "time.interval": {
    key: "time.interval",
    name: "Time Interval",
    zIndex: 100,
    Form: IntervalForm,
    Symbol: Interval,
    Preview: Interval,
    defaultProps: () => ({
      duration: 1000,
    }),
  },
  "time.schedule": {
    key: "time.schedule",
    name: "Schedule",
    zIndex: 100,
    Form: ScheduleForm,
    Symbol: Schedule,
    Preview: Schedule,
    defaultProps: () => ({
      interval: 1000,
    }),
  },
};
