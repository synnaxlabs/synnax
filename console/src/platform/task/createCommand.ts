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
import { createOpenTab } from "@/platform/task/useOpenTab";

export interface CreateCommandParams {
  key: string;
  name: string;
  icon: Icon.ReactElement;
  type: string;
}

const useVisible = () => Access.useCreateGranted(task.TYPE_ONTOLOGY_ID);

/** Creates a palette command that opens a blank task form of the given type. */
export const createCommand = ({
  type,
  ...rest
}: CreateCommandParams): Command.Command =>
  Command.create({ ...rest, useOnSelect: createOpenTab(type), useVisible });
