// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { errors } from "@synnaxlabs/x";

import { restart, stop } from "@/feature/embedded/supervisor";
import { type Version } from "@/platform/version";

/**
 * Stops the embedded Core while an update installs. An installer cannot replace the
 * executable of a Core that runs. The Core starts again when the install fails.
 */
export const installMiddleware: Version.InstallMiddleware = async (install) => {
  await stop();
  try {
    await install();
  } catch (err) {
    await restart();
    throw errors.fromUnknown(err);
  }
};
