// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";

const safe = (name: string): string => channel.escapeInvalidName(name, true);

/** @returns the name of the index channel of a message a read task decodes. */
export const indexName = (device: string, message: string): string =>
  `${safe(device)}_${safe(message)}_time`;

/** @returns the name of the channel a read task writes a field to. */
export const fieldChannelName = (device: string, message: string, field: string) =>
  `${safe(device)}_${safe(message)}_${safe(field)}`;

/** @returns the name of the index channel of a message's command channels. */
export const commandIndexName = (device: string, message: string): string =>
  `${safe(device)}_${safe(message)}_cmd_time`;

/** @returns the name of the command channel a write task encodes a field from. */
export const commandChannelName = (device: string, message: string, field: string) =>
  `${fieldChannelName(device, message, field)}_cmd`;

/** @returns the name of the virtual channel a read task writes raw frames to. */
export const rawName = (device: string): string => `${safe(device)}_raw`;
