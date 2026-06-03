// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Instrumentation } from "@synnaxlabs/alamos";
import { errors } from "@synnaxlabs/x";

import { type Context, type Middleware } from "@/middleware";

export const middleware =
  (instrumentation: Instrumentation): Middleware =>
  async (context, next) => {
    if (context.role === "client") instrumentation.T.propagate(context.params);
    try {
      const res = await instrumentation.T.trace(
        context.target,
        "debug",
        async (span): Promise<Context> => {
          try {
            return await next(context);
          } catch (err) {
            if (err instanceof Error) span.recordError(err);
            throw errors.toError(err);
          }
        },
      );
      log(context, instrumentation, null);
      return res;
    } catch (err) {
      const e = errors.toError(err);
      log(context, instrumentation, e);
      throw e;
    }
  };

const log = (
  context: Context,
  instrumentation: Instrumentation,
  err: Error | null,
): void =>
  err != null
    ? instrumentation.L.error(
        `${context.target} ${context.protocol} failed: ${err.message}`,
      )
    : instrumentation.L.debug(`${context.target} ${context.protocol}`);
