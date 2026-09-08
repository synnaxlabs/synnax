// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// The CI media check (src/util/checks/media.ts) rebuilds these URLs from island props
// to verify the assets exist on the CDN. Keep it dependency-free.

export const CDN_ROOT = "https://synnax.nyc3.cdn.digitaloceanspaces.com/docs";

export const mediaURL = (id: string, extension: string, theme?: string): string =>
  `${CDN_ROOT}/${id}${theme != null ? `-${theme}` : ""}.${extension}`;
