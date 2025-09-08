import { Form as IntervalForm } from "@/arc/stage/time/Form";
import { Interval } from "@/arc/stage/time/Interval";
import { Schedule } from "@/arc/stage/time/Schedule";
import { Form as ScheduleForm } from "@/arc/stage/time/ScheduleForm";
import { type types } from "@/arc/stage/types";

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
