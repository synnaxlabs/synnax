// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import cfg, { reactCompiled } from "@synnaxlabs/eslint-config";
import type { Linter } from "eslint";

export default [
  ...cfg,
  reactCompiled,
  { ignores: ["examples"] },
  // Off until this spec tests through configureStore instead of internal files.
  { files: ["src/state.spec.ts"], rules: { "synnax/barrel-imports": "off" } },
] as Linter.Config[];
