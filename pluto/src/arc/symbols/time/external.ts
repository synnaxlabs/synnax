import { Form as IntervalForm } from "@/arc/symbols/time/Form";
import { Interval } from "@/arc/symbols/time/Interval";
import { Schedule } from "@/arc/symbols/time/Schedule";
import { Form as ScheduleForm } from "@/arc/symbols/time/ScheduleForm";
import { type types } from "@/arc/symbols/types";

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
