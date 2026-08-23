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

/** How strongly a writer claims a channel. The highest authority holds control. */
export type Authority = control.Authority;
/** The highest authority. A writer holding it cannot be displaced. */
export const ABSOLUTE_AUTHORITY = control.ABSOLUTE_AUTHORITY;
/** The lowest authority. Any other writer displaces it. */
export const ZERO_AUTHORITY = control.ZERO_AUTHORITY;
/** Control of one channel moving between writers, or being released. */
export type Transfer = control.Transfer<typeof channel.keyZ>;
/** Who holds control of a channel, and at what authority. */
export interface State extends control.State<typeof channel.keyZ> {}
/** A named party that can hold control. */
export interface Subject extends control.Subject {}
/** Zod schema for {@link State}. */
export const stateZ = control.stateZ(z.number());

/**
 * A control state stored under the channel it controls, which is how the client caches
 * it.
 */
export const keyedStateZ = stateZ.transform((s) => ({ key: s.resource, ...s }));
export interface KeyedState extends z.infer<typeof keyedStateZ> {}

/** Renders a {@link Transfer} as readable text, for logs and status messages. */
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
