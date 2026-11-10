// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Version } from "@/util/version";

/**
 * The docs version this build describes, set at build time from `DOCS_VERSION`. Null
 * for an unversioned build, such as a dev server or a pull request preview.
 */
export const VERSION: Version | null = __DOCS_VERSION__;
