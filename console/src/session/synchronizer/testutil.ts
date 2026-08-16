// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Action } from "@reduxjs/toolkit";

import { type Synchronizers } from "@/session/synchronizer/create";

/**
 * Narrows a registry to the named synchronizer so a spec can mount it without
 * its siblings.
 * @throws {Error} if the registry holds no synchronizer under the name.
 */
export const pickSynchronizer = <S, A extends Action>(
  synchronizers: Synchronizers<S, A>,
  name: string,
): Synchronizers<S, A> => {
  const synchronizer = synchronizers.find((sync) => sync.name === name);
  if (synchronizer == null) throw new Error(`no synchronizer named ${name}`);
  return [synchronizer];
};
