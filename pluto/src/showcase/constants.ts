// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export const DISPLAYS = [
  "breadcrumb",
  "button",
  "divider",
  "flex",
  "header",
  "input",
  "note",
  "select",
  "tag",
  "text",
] as const;

export type Display = (typeof DISPLAYS)[number];
