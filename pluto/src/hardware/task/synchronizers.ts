// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { z } from "zod/v4";

import { Sync } from "@/sync";

export const useCommandSynchronizer = (
  onCommandUpdate: (command: task.Command) => void,
): void =>
  Sync.useParsedListener(task.COMMAND_CHANNEL_NAME, task.commandZ, onCommandUpdate);

export const useStatusSynchronizer = <StatusData extends z.ZodTypeAny>(
  onStatusUpdate: (status: task.Status<StatusData>) => void,
  statusDataZ: StatusData = z.unknown() as unknown as StatusData,
): void =>
  Sync.useParsedListener(
    task.STATUS_CHANNEL_NAME,
    task.statusZ(statusDataZ),
    onStatusUpdate,
  );

export const useSetSynchronizer = (onSet: (key: task.Key) => void): void =>
  Sync.useParsedListener(task.SET_CHANNEL_NAME, task.keyZ, onSet);

export const useDeleteSynchronizer = (onDelete: (key: task.Key) => void): void =>
  Sync.useParsedListener(task.DELETE_CHANNEL_NAME, task.keyZ, onDelete);
