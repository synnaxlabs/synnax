// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { Access, type Icon } from "@synnaxlabs/pluto";

import { Command } from "@/platform/command";
import { type UseCreate } from "@/platform/task/useCreate";

export interface CreateCommandParams {
  key: string;
  name: string;
  icon: Icon.ReactElement;
  useOnSelect: UseCreate;
}

const useVisible = () => Access.useCreateGranted(task.TYPE_ONTOLOGY_ID);

/** Creates a palette command that creates a task of the given type and opens it. */
export const createCommand = (params: CreateCommandParams): Command.Command =>
  Command.create({ ...params, useVisible });
