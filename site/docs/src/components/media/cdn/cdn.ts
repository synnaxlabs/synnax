// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// The CI media check imports this module in Node. Keep it free of dependencies.

export const ROOT = "https://synnax.nyc3.cdn.digitaloceanspaces.com/docs";

export const url = (id: string, extension: string, theme?: string): string =>
  `${ROOT}/${id}${theme != null ? `-${theme}` : ""}.${extension}`;
