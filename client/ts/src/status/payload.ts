// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/x";
import { z } from "zod";

import { type ontology } from "@/ontology";

export const keyZ = z.string();
export type Key = z.infer<typeof keyZ>;

export type Params = Key | Key[];

export const statusZ = status.statusZ;

export const newZ = <DetailsSchema extends z.ZodType = z.ZodNever>(
  detailsSchema?: DetailsSchema,
) => statusZ(detailsSchema).omit({ labels: true }).partial({ key: true });

export type New<DetailsSchema extends z.ZodType = z.ZodNever> = z.input<
  ReturnType<typeof newZ<DetailsSchema>>
>;

export type Status<Details = never> = status.Status<Details>;

export const SET_CHANNEL_NAME = "sy_status_set";
export const DELETE_CHANNEL_NAME = "sy_status_delete";

export const ontologyID = (key: Key): ontology.ID => ({ type: "status", key });
