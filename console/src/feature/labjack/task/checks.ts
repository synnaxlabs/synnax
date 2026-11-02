// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type z } from "zod";

import { type Model, T8_MODEL } from "@/feature/labjack/device/types";
import { claims } from "@/feature/labjack/task/ports";
import { type Channel } from "@/feature/labjack/task/types";
import { Task } from "@/platform/task";

/**
 * Checks that no two channels of a task config use one terminal of the model. On a T4,
 * a channel on AIN4 and one on DIO4 both use FIO4.
 */
export const uniqueTerminalsZ = (model: Model): z.ZodType =>
  Task.check<{ channels: Channel[] }>(({ channels }, report) =>
    Task.Port.reportConflicts(claims(channels, model), channels, report),
  );

/** The highest sample rate of a T8 task with thermocouples. */
export const T8_MAX_THERMOCOUPLE_RATE = 250;

/**
 * Checks that a T8 task with an enabled thermocouple samples at 250 Hz or less. Above
 * that rate, the CJC sensors of a T8 read no value.
 */
export const thermocoupleRateZ = (model: Model): z.ZodType =>
  Task.check<{ sampleRate: number; channels: Channel[] }>(
    ({ sampleRate, channels }, report) => {
      if (model !== T8_MODEL || sampleRate <= T8_MAX_THERMOCOUPLE_RATE) return;
      if (!channels.some(({ disabled, type }) => !disabled && type === "thermocouple"))
        return;
      report(
        ["sampleRate"],
        `A T8 task with thermocouples samples at ${T8_MAX_THERMOCOUPLE_RATE} Hz or less`,
      );
    },
  );
