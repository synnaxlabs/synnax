// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, TimeStamp } from "@synnaxlabs/x";
import { z } from "zod";

import { ontology } from "@/ontology";
import { type New, type Status } from "@/status/types.gen";

export const keyZ = z.string();
export type Key = z.infer<typeof keyZ>;

/**
 * Builds a server-side status entity from creation parameters, filling in a generated
 * key, an empty name, and the current time when they are not provided.
 */
export const create = <Details extends z.ZodType = z.ZodNever>(
  spec: New<Details>,
): Status<Details> =>
  ({
    key: id.create(),
    name: "",
    time: TimeStamp.now(),
    ...spec,
  }) as unknown as Status<Details>;

export const SET_CHANNEL_NAME = "sy_status_set";
export const DELETE_CHANNEL_NAME = "sy_status_delete";

export const ontologyID = ontology.createIDFactory<Key>("status");
export const TYPE_ONTOLOGY_ID = ontologyID("");
