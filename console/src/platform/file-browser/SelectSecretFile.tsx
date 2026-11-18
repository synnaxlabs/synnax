// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack, status, type Synnax as Client } from "@synnaxlabs/client";
import { useAsyncEffect } from "@synnaxlabs/lyra/hooks";
import { type Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Synnax } from "@synnaxlabs/pluto";
import { unique } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { list } from "@/platform/file-browser/list";

type State =
  | { variant: "loading" }
  | { variant: "none" }
  | { variant: "error"; status: status.Status }
  | { variant: "success"; names: string[] };

const listNames = async (client: Client, rack: rack.Key): Promise<State> => {
  const { secretsDir } = await list(client, rack);
  if (secretsDir === "") return { variant: "none" };
  const { entries } = await list(client, rack, secretsDir);
  return {
    variant: "success",
    names: entries.filter((e) => e.kind === "file").map((e) => e.name),
  };
};

export interface SelectSecretFileProps extends Input.Control<string> {
  /** The rack whose secrets folder the select lists. */
  rack: rack.Key;
  /** The text of an empty value, such as "None" for an optional file. */
  placeholder?: string;
}

/**
 * Selects a file in the secrets folder of a rack by its name. An empty value names no
 * file.
 */
export const SelectSecretFile = ({
  rack,
  value,
  onChange,
  placeholder = "Select a file",
}: SelectSecretFileProps): ReactElement => {
  const client = Synnax.use();
  const [state, setState] = useState<State>({ variant: "loading" });
  useAsyncEffect(
    async (signal) => {
      setState({ variant: "loading" });
      if (client == null) return;
      let next: State;
      try {
        next = await listNames(client, rack);
      } catch (e) {
        next = {
          variant: "error",
          status: status.fromException(e, "Failed to list files"),
        };
      }
      if (!signal.aborted) setState(next);
    },
    [client, rack],
  );
  const names = state.variant === "success" ? state.names : [];
  const options = unique.unique([...names, value]).filter((n) => n !== "");
  let emptyContent: ReactElement | string = "Listing files";
  if (state.variant === "error")
    emptyContent = <Status.Summary level="small" status={state.status} />;
  else if (state.variant === "success")
    emptyContent =
      names.length === 0 ? "The secrets folder is empty" : "No matching files";
  return (
    <Select.Simple<string>
      resourceName="file"
      value={value === "" ? undefined : value}
      onChange={(next: string | null) => onChange(next ?? "")}
      allowNone
      disabled={state.variant === "none"}
      emptyContent={emptyContent}
      triggerProps={{
        placeholder:
          state.variant === "none" ? "The rack has no secrets folder" : placeholder,
      }}
      grow
    >
      {options.map((n) => (
        <Select.Item key={n} itemKey={n}>
          {n}
        </Select.Item>
      ))}
    </Select.Simple>
  );
};
