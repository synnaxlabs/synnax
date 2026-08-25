// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Command } from "@/platform/command";

/**
 * Returns the command a domain offers under the name, so a spec can read its gate
 * without depending on the order of the domain's COMMANDS.
 * @throws {Error} if no command carries the name.
 */
export const findCommand = (
  commands: Command.Command[],
  name: string,
): Command.Command => {
  const command = commands.find(({ commandName }) => commandName === name);
  if (command == null) throw new Error(`no command named ${name}`);
  return command;
};
