// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status as xStatus } from "@synnaxlabs/x";
import { z } from "zod";

import { label } from "@/label";
import { type ontology } from "@/ontology";
import { nullableArrayZ } from "@/util/zod";

export const keyZ = z.string();
export type Key = z.infer<typeof keyZ>;

export type Params = Key | Key[];

// The Status type combines a name with the base status from x/status
export const statusZ = xStatus.statusZ().extend({
  labels: nullableArrayZ(label.labelZ),
});

export const newZ = statusZ.omit({ labels: true }).partial({ key: true });

export interface New extends z.input<typeof newZ> {}

export interface Status extends z.infer<typeof statusZ> {}

export const SET_CHANNEL_NAME = "sy_status_set";
export const DELETE_CHANNEL_NAME = "sy_status_delete";

export const ontologyID = (key: Key): ontology.ID => ({ type: "status", key });
