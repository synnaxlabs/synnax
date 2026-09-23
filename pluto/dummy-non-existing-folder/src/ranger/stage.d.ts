import { type ranger } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type CrudeTimeRange, type NumericTimeRange, TimeStamp } from "@synnaxlabs/x";
export declare const STAGES: readonly ["to_do", "in_progress", "completed"];
export type Stage = (typeof STAGES)[number];
export declare const sortByStage: (a: ranger.Range, b: ranger.Range) => number;
/** @returns the stage `timeRange` is in at `now`. */
export declare const getStage: (timeRange: CrudeTimeRange, now?: TimeStamp) => Stage;
export declare const STAGE_ICONS: Record<Stage, Icon.FC>;
export declare const STAGE_NAMES: Record<Stage, string>;
/**
 * Returns the range with the timestamps that put it in `stage` as of now: to do moves
 * a past start to the end, in progress stamps a future start and clears a past end,
 * and completed stamps a future end.
 */
export declare const moveToStage: (value: NumericTimeRange, stage: Stage) => NumericTimeRange;
//# sourceMappingURL=stage.d.ts.map