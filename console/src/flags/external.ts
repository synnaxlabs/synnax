// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Owner: Emiliano Bonilla. Umbrella: SY-4970. Removed in 0.63, when ICDs ship. No task
// reads an ICD yet, so Desktop builds leave it out.
export const icd = IS_DEV || import.meta.env.VITE_FLAGS_ENABLED === "true";
