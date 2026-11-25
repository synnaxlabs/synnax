// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { enumeration, schematic } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Input } from "@synnaxlabs/lyra/input";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Status } from "@synnaxlabs/lyra/status";
import { type ReactElement, useState } from "react";

import { Enumeration } from "@/enumeration";
import type { bind } from "@/schematic/node/symbol/bind";

/** Sendable is a command with the value that it writes and where it writes it. */
export interface Sendable {
  command: schematic.symbol.Command;
  value: number;
  target: bind.Target;
}

/** Ask is a command that asks for its value, with where it writes and its state. */
export interface Ask extends Omit<Sendable, "value"> {
  state: schematic.symbol.State;
}

/** Request is what a click asks the user: which of options to send, or a value. */
export type Request = { type: "choose"; options: Sendable[] } | ({ type: "ask" } & Ask);

export interface PromptProps {
  request: Request;
  onSend: (s: Sendable) => void;
}

/**
 * Prompt is the content of a dialog that answers request. An ask command gets a menu of
 * the entries of its state's enumeration, or a number field for any other state.
 */
export const Prompt = ({ request, onSend }: PromptProps): ReactElement => {
  if (request.type === "choose")
    return (
      <Options
        aria-label="Commands"
        options={request.options.map((o) => ({
          key: o.command.key,
          name: o.command.name,
          onSelect: () => onSend(o),
        }))}
      />
    );
  const { command, target, state } = request;
  if (state.type === "enumeration" && state.enumeration != null)
    return (
      <Entries
        command={command}
        enumeration={state.enumeration}
        onSend={(value) => onSend({ command, target, value })}
      />
    );
  return (
    <Field command={command} onSend={(value) => onSend({ command, target, value })} />
  );
};

interface Option {
  key: string;
  name: string;
  onSelect: () => void;
}

interface OptionsProps {
  "aria-label": string;
  options: Option[];
}

const Options = ({ options, ...rest }: OptionsProps): ReactElement => (
  <Menu.Dialog {...rest}>
    <Menu.Menu onChange={Object.fromEntries(options.map((o) => [o.key, o.onSelect]))}>
      {options.map(({ key, name }) => (
        <Menu.Item key={key} itemKey={key}>
          {name}
        </Menu.Item>
      ))}
    </Menu.Menu>
  </Menu.Dialog>
);

interface AskProps {
  command: schematic.symbol.Command;
  onSend: (value: number) => void;
}

const Entries = ({
  command,
  enumeration,
  onSend,
}: AskProps & { enumeration: enumeration.Key }): ReactElement => {
  const res = Enumeration.useResult({ key: enumeration });
  if (res.variant !== "success")
    return (
      <Menu.Dialog itemless aria-label={command.name}>
        <Status.Summary status={res.status} />
      </Menu.Dialog>
    );
  return (
    <Options
      aria-label={command.name}
      options={res.data.entries.map(({ key, name, value }) => ({
        key,
        name,
        onSelect: () => onSend(value),
      }))}
    />
  );
};

// parse returns the number in the text of a number input, which is empty for any text
// that is not a number, or NaN for none.
const parse = (text: string): number => (text.trim() === "" ? NaN : Number(text));

const Field = ({ command, onSend }: AskProps): ReactElement => {
  const [text, setText] = useState("");
  const value = parse(text);
  const send = (): void => {
    if (!Number.isNaN(value)) onSend(value);
  };
  return (
    <Menu.Dialog itemless aria-label={command.name}>
      <Input.Text
        type="number"
        value={text}
        onChange={setText}
        onKeyDown={(e) => {
          if (e.key === "Enter") send();
        }}
        autoFocus
      >
        <Button.Button variant="filled" disabled={Number.isNaN(value)} onClick={send}>
          {command.name}
        </Button.Button>
      </Input.Text>
    </Menu.Dialog>
  );
};
