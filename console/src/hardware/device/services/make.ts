// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.import { type LabJack } from "./labjack";

import { z } from "zod";

import { LabJack } from "@/hardware/labjack";
import { NI } from "@/hardware/ni";
import { OPC } from "@/hardware/opc";

export const makeZ = z.enum([NI.Device.MAKE, LabJack.Device.MAKE, OPC.Device.MAKE]);
export type Make = z.infer<typeof makeZ>;
