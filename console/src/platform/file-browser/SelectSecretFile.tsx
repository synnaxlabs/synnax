// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack, status } from "@synnaxlabs/client";
import { Divider } from "@synnaxlabs/lyra/divider";
import { useAsyncEffect } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import type { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { unique } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { useSession } from "@/platform/file-browser/Provider";
import type { Session } from "@/platform/file-browser/session";

type State =
  | { variant: "loading" }
  | { variant: "none" }
  | { variant: "error"; status: status.Status }
  | { variant: "success"; names: string[] };

const listNames = async (session: Session): Promise<State> => {
  const { secretsDir } = await session.list();
  if (secretsDir === "") return { variant: "none" };
  const { entries } = await session.list(secretsDir);
  return {
    variant: "success",
    names: entries.filter((e) => e.kind === "file").map((e) => e.name),
  };
};

// A file name never holds a slash, so this key names no file.
const NONE = "/";

export interface SelectSecretFileProps extends Input.Control<string> {
  /** The rack whose secrets folder the select lists. */
  rack: rack.Key;
  /** Whether no file is a valid value. It adds a "No file" option. */
  allowNone?: boolean;
}

/**
 * Selects a file in the secrets folder of a rack by its name. An empty value names no
 * file.
 */
export const SelectSecretFile = ({
  rack,
  value,
  onChange,
  allowNone = false,
}: SelectSecretFileProps): ReactElement => {
  const session = useSession(rack);
  const [state, setState] = useState<State>({ variant: "loading" });
  useAsyncEffect(
    async (signal) => {
      setState({ variant: "loading" });
      if (session == null) return;
      let next: State;
      try {
        next = await listNames(session);
      } catch (e) {
        next = {
          variant: "error",
          status: status.fromException(e, "Failed to list files"),
        };
      }
      if (!signal.aborted) setState(next);
    },
    [session],
  );
  const names = state.variant === "success" ? state.names : [];
  const options = unique.unique([...names, value]).filter((n) => n !== "");
  const folderMissing = state.variant === "none";
  const empty = allowNone && !folderMissing ? NONE : undefined;
  const selected = value === "" ? empty : value;
  let emptyContent: ReactElement | string = "Listing files";
  if (state.variant === "error")
    emptyContent = <Status.Summary level="small" status={state.status} />;
  else if (state.variant === "success")
    emptyContent =
      names.length === 0 ? "The secrets folder is empty" : "No matching files";
  return (
    <Select.Simple<string>
      resourceName="file"
      value={selected}
      onChange={(next: string | null) =>
        onChange(next == null || next === NONE ? "" : next)
      }
      disabled={folderMissing}
      emptyContent={emptyContent}
      triggerProps={{
        placeholder: folderMissing
          ? "The Driver has no secrets folder"
          : "Select a file",
      }}
      footer={
        allowNone && (
          <>
            <Divider.Divider x />
            <Select.Item itemKey={NONE} textValue="No file">
              <Icon.None />
              No file
            </Select.Item>
          </>
        )
      }
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
