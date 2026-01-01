// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { type bounds } from "@/spatial";

export const authorityZ = z.int().min(0).max(255);
export type Authority = z.infer<typeof authorityZ>;

export const ABSOLUTE_AUTHORITY: Authority = 255;
export const ZERO_AUTHORITY: Authority = 0;

export const AUTHORITY_BOUNDS: bounds.Bounds<Authority> = {
  lower: ZERO_AUTHORITY,
  upper: ABSOLUTE_AUTHORITY + 1,
};

export const subjectZ = z.object({
  name: z.string(),
  key: z.string(),
});

export interface Subject {
  name: string;
  key: string;
}

export const stateZ = <R extends z.ZodType>(resource: R) =>
  z.object({
    subject: subjectZ,
    resource,
    authority: authorityZ,
  });

export interface State<R> {
  subject: Subject;
  resource: R;
  authority: Authority;
}

export const filterTransfersByChannelKey =
  <R>(...resources: R[]) =>
  (transfers: Transfer<R>[]): Transfer<R>[] =>
    transfers.filter((t) => {
      let ok = false;
      if (t.to != null) ok = resources.includes(t.to.resource);
      if (t.from != null && !ok) ok = resources.includes(t.from.resource);
      return ok;
    });

interface Release<R> {
  from: State<R>;
  to?: null;
}

export const releaseZ = <R extends z.ZodType>(resource: R) =>
  z.object({
    from: stateZ(resource),
    to: z.null(),
  });

interface Acquire<R> {
  from?: null;
  to: State<R>;
}

export const acquireZ = <R extends z.ZodType>(resource: R) =>
  z.object({
    from: z.null(),
    to: stateZ(resource),
  });

export type Transfer<R> =
  | {
      from: State<R>;
      to: State<R>;
    }
  | Release<R>
  | Acquire<R>;

export const transferZ = <R extends z.ZodType>(resource: R) =>
  z.union([
    releaseZ(resource),
    acquireZ(resource),
    z.object({
      from: stateZ(resource),
      to: stateZ(resource),
    }),
  ]);
