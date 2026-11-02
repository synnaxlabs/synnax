// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const COMPUTER = /Macintosh|Windows/;

const MOBILE = /iPhone|iPad|iPod|Android|Mobile/;

/**
 * installable reads from a browser's user agent whether it runs on a computer that
 * Synnax Desktop installs on: a Mac or a Windows PC, not a phone or a tablet.
 */
export const installable = (userAgent: string): boolean =>
  COMPUTER.test(userAgent) && !MOBILE.test(userAgent);
