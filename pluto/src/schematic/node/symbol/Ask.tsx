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
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Input } from "@synnaxlabs/lyra/input";
import { Menu } from "@synnaxlabs/lyra/menu";
import { type bounds, state } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { Enumeration } from "@/enumeration";

interface EntriesProps {
  enumeration: enumeration.Key;
  onSend: (value: number) => void;
}

// An enumeration that fails to load has no entries.
const Entries = ({ enumeration, onSend }: EntriesProps): ReactElement => {
  const res = Enumeration.useResult({ key: enumeration });
  const entries = res.variant === "success" ? res.data.entries : [];
  return (
    <Menu.Menu
      onChange={Object.fromEntries(entries.map((e) => [e.key, () => onSend(e.value)]))}
    >
      {entries.map((e) => (
        <Menu.Item key={e.key} itemKey={e.key}>
          {e.name}
        </Menu.Item>
      ))}
    </Menu.Menu>
  );
};

interface FieldProps {
  bounds: bounds.Bounds;
  initial: number;
  onSend: (value: number) => void;
}

const Field = ({ bounds, initial, onSend }: FieldProps): ReactElement => {
  const [draft, setDraft] = useState(initial);
  return (
    <Flex.Box x align="center" gap="small">
      <Input.Numeric value={draft} onChange={setDraft} bounds={bounds} autoFocus />
      <Button.Button variant="filled" onClick={() => onSend(draft)}>
        Send
      </Button.Button>
    </Flex.Box>
  );
};

export interface AskProps {
  command: schematic.symbol.Command;
  /** state is the state of command, with the bindings of the node applied. */
  state: schematic.symbol.State;
  /** value is the value of state, or undefined when it has none. */
  value?: number;
  anchor: HTMLElement;
  onSend: (value: number) => void;
  onClose: () => void;
}

/**
 * Ask asks for the value that command writes: an entry of an enumeration state, or a
 * number within the bounds of a number state. A typed number writes only on Send.
 */
export const Ask = ({
  command,
  state: s,
  value,
  anchor,
  onSend,
  onClose,
}: AskProps): ReactElement => (
  <Dialog.Frame
    anchor={anchor}
    visible
    onVisibleChange={(next) => {
      if (!state.resolve(next, true)) onClose();
    }}
  >
    {s.type === "enumeration" ? (
      <Menu.Dialog aria-label={command.name}>
        {s.enumeration != null && (
          <Entries enumeration={s.enumeration} onSend={onSend} />
        )}
      </Menu.Dialog>
    ) : (
      <Menu.Dialog itemless aria-label={command.name}>
        <Field bounds={s.bounds} initial={value ?? s.bounds.lower} onSend={onSend} />
      </Menu.Dialog>
    )}
  </Dialog.Frame>
);
