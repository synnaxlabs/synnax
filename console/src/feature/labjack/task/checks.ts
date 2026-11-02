// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type z } from "zod";

import { type Device } from "@/feature/labjack/device";
import { claims } from "@/feature/labjack/task/ports";
import { type Channel } from "@/feature/labjack/task/types";
import { Task } from "@/platform/task";

/**
 * Checks that no two channels of a task config use one terminal of the model. On a T4,
 * a channel on AIN4 and one on DIO4 both use FIO4.
 */
export const uniqueTerminalsZ = (model: Device.Model): z.ZodType =>
  Task.check<{ channels: Channel[] }>(({ channels }, report) =>
    Task.Port.reportConflicts(claims(channels, model), channels, report),
  );
