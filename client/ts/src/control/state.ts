// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { control } from "@synnaxlabs/x";
import { z } from "zod";

import { type channel } from "@/channel";
import { keyZ } from "@/channel/types.gen";

export type Authority = control.Authority;
export const ABSOLUTE_AUTHORITY = control.ABSOLUTE_AUTHORITY;
export const ZERO_AUTHORITY = control.ZERO_AUTHORITY;
export type Transfer = control.Transfer<typeof channel.keyZ>;
export interface State extends control.State<typeof channel.keyZ> {}
export interface Subject extends control.Subject {}
export const stateZ = control.stateZ(z.number());

/**
 * A control state stored under the channel it controls, which is how the client
 * caches it.
 */
export const keyedStateZ = stateZ.transform((s) => ({ key: s.resource, ...s }));
export interface KeyedState extends z.infer<typeof keyedStateZ> {}

export const transferString = (t: Transfer): string => {
  const fromResource = t.from?.resource;
  const toResource = t.to?.resource;
  if (t.to == null) return `${fromResource} - ${t.from?.subject.name} -> released`;
  if (t.from == null)
    return `${toResource} - released -> ${
      t.to.subject.name
    } (${t.to.authority.toString()})`;
  return `${toResource} - ${t.from.subject.name} -> ${
    t.to.subject.name
  } (${t.to.authority.toString()})`;
};

export const updateZ = z.object({
  transfers: z.array(control.transferZ(keyZ)),
});

export interface Update extends z.infer<typeof updateZ> {}
