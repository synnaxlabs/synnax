// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { Data } from "@/schematic/edge/data";
import { Electric } from "@/schematic/edge/electric";
import { Hydraulic } from "@/schematic/edge/hydraulic";
import { Jacketed } from "@/schematic/edge/jacketed";
import { Pipe } from "@/schematic/edge/pipe";
import { Pneumatic } from "@/schematic/edge/pneumatic";
import { Secondary } from "@/schematic/edge/secondary";

export const configZ = z.discriminatedUnion("variant", [
  Pipe.configZ,
  Electric.configZ,
  Secondary.configZ,
  Jacketed.configZ,
  Hydraulic.configZ,
  Pneumatic.configZ,
  Data.configZ,
]);
export type Config = z.infer<typeof configZ>;
