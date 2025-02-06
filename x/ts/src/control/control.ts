// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export class Authority extends Number {
  static readonly Absolute = 255;
  static readonly Default = 1;

  static readonly z = z.union([
    z.instanceof(Authority),
    z
      .number()
      .int()
      .min(0)
      .max(255)
      .transform((n) => new Authority(n)),
    z.instanceof(Number).transform((n) => new Authority(n)),
  ]);
}

export const subjectZ = z.object({
  name: z.string(),
  key: z.string(),
});

export interface Subject {
  name: string;
  key: string;
}

export const stateZ = <T extends z.ZodTypeAny>(r: T) =>
  z.object({
    subject: subjectZ,
    resource: r,
    authority: Authority.z,
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

interface Acquire<R> {
  from?: null;
  to: State<R>;
}

export type Transfer<R> =
  | {
      from: State<R>;
      to: State<R>;
    }
  | Release<R>
  | Acquire<R>;
