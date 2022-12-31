// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Middleware } from "../middleware";

export const logMiddleware = (): Middleware => {
  return async (md, next) => {
    console.log(JSON.stringify(md, undefined, 2));
    const err = await next(md);
    if (err != null) {
      console.log(err);
    }
    return err;
  };
};
