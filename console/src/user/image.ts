// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** The maximum width or height, in pixels, of a stored avatar. */
export const MAX_AVATAR_DIMENSION = 256;

/**
 * The maximum size, in bytes, of the encoded avatar data URI. Mirrors the server-side
 * cap enforced on the user record.
 */
export const MAX_AVATAR_BYTES = 65536;

const QUALITY_STEPS = [0.8, 0.6, 0.4];

/**
 * Downscales the given image file to a square-bounded WebP data URI suitable for use as
 * a user avatar. The image is scaled so its longest edge is at most
 * {@link MAX_AVATAR_DIMENSION} pixels and re-encoded at decreasing quality until it fits
 * within {@link MAX_AVATAR_BYTES}.
 *
 * @throws {Error} if the file cannot be decoded as an image, a canvas context cannot be
 * acquired, or the image cannot be compressed below the size cap.
 */
export const fileToAvatarDataURI = async (file: File): Promise<string> => {
  const bitmap = await createImageBitmap(file);
  try {
    const scale = Math.min(
      1,
      MAX_AVATAR_DIMENSION / Math.max(bitmap.width, bitmap.height),
    );
    const width = Math.max(1, Math.round(bitmap.width * scale));
    const height = Math.max(1, Math.round(bitmap.height * scale));
    const canvas = document.createElement("canvas");
    canvas.width = width;
    canvas.height = height;
    const ctx = canvas.getContext("2d");
    if (ctx == null) throw new Error("failed to acquire a 2D canvas context");
    ctx.drawImage(bitmap, 0, 0, width, height);
    for (const quality of QUALITY_STEPS) {
      const uri = canvas.toDataURL("image/webp", quality);
      if (uri.length <= MAX_AVATAR_BYTES) return uri;
    }
    throw new Error(
      "image is too detailed to use as an avatar; try a smaller or simpler image",
    );
  } finally {
    bitmap.close();
  }
};
