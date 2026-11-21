// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type vercel from "@astrojs/vercel";
import { CDN } from "@synnaxlabs/site-common/cdn";

/**
 * Vercel resizes and encodes images when a reader requests them, so the build encodes
 * none. It serves AVIF or WebP by the browser's Accept header.
 */
export const IMAGES = {
  sizes: [640, 960, 1280, 1920, 2560],
  domains: [new URL(CDN).hostname],
  formats: ["image/avif", "image/webp"],
} satisfies NonNullable<Parameters<typeof vercel>[0]>["imagesConfig"];

/** Builds image URLs as the Vercel adapter does, without encoding anything. */
export const VERCEL_IMAGE_SERVICE = {
  entrypoint: "@astrojs/vercel/build-image-service",
  config: IMAGES,
};
