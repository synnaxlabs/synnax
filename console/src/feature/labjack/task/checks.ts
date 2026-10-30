// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { type Model } from "@/feature/labjack/device/types";
import { claims, conflicts, terminalOf } from "@/feature/labjack/task/ports";
import { type Channel } from "@/feature/labjack/task/types";

/**
 * Checks that no two channels of a task config use one terminal of the model. On a T4,
 * a channel on AIN4 and one on DIO4 both use FIO4.
 */
export const uniqueTerminalsZ = (model: Model): z.ZodType =>
  z.custom<{ channels: Channel[] }>().check(({ value, issues }) => {
    const taken = claims(value.channels, model);
    value.channels.forEach(({ port, disabled }, i) => {
      if (disabled) return;
      const name = terminalOf(model, port);
      if (!conflicts(taken.get(name) ?? [])) return;
      issues.push({
        code: "custom",
        path: ["channels", i, "port"],
        message: `Another channel uses ${name}`,
        input: value,
      });
    });
  });
