// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import { Divider } from "@synnaxlabs/lyra/divider";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Schematic } from "@synnaxlabs/pluto";
import type { ReactElement } from "react";

import { Spec } from "@/feature/schematic/symbol/spec";
import { CSS } from "@/platform/css";

const path = Schematic.Symbol.formPath;

type Interaction = schematic.symbol.Interaction;

// WHOLE_DRAWING is the key of the whole drawing, which a select cannot hold as
// undefined.
const WHOLE_DRAWING = "";

/** @returns the name of the target of a click: a part, or the whole drawing. */
export const targetName = (parts: schematic.symbol.Part[], part?: string): string =>
  part == null
    ? "Whole drawing"
    : (parts.find((p) => p.key === part)?.name ?? "Missing part");

interface CommandButtonsProps {
  value: string | undefined;
  onChange: (key: string) => void;
  commands: schematic.symbol.Command[];
}

const CommandButtons = ({
  value,
  onChange,
  commands,
}: CommandButtonsProps): ReactElement => (
  <Select.Buttons<string>
    value={value}
    onChange={onChange}
    size="small"
    variant="outlined"
  >
    {commands.map((c) => (
      <Select.Item<string> key={c.key} itemKey={c.key} size="small">
        {c.name}
      </Select.Item>
    ))}
  </Select.Buttons>
);

export interface ClickInspectorProps {
  /** click is the key of the click to edit. */
  click: string;
}

/** ClickInspector edits the target of one click, and the command that it sends. */
export const ClickInspector = ({
  click: key,
}: ClickInspectorProps): ReactElement | null => {
  const click = Form.useFieldValue(path.data.interactions[key], { optional: true });
  const parts = Form.useFieldValue(path.data.parts);
  const commands = Form.useFieldValue(path.data.commands);
  const interactions = Form.useFieldValue(path.data.interactions);
  const { set } = Form.useContext();
  if (click == null) return null;
  const update = (next: Interaction) => set(path.data.interactions[key], next);
  const targets = [click.part, ...Spec.freeTargets({ parts, interactions })];
  return (
    <Flex.Box y gap="medium" className={CSS.B("schematic-inspector")}>
      <Text.Text level="h5">{targetName(parts, click.part)}</Text.Text>
      <Divider.Divider x />
      <Input.Item label="Target">
        <Select.Buttons<string>
          value={click.part ?? WHOLE_DRAWING}
          onChange={(t: string) =>
            update({ ...click, part: t === WHOLE_DRAWING ? undefined : t })
          }
          size="small"
          variant="outlined"
        >
          {targets.map((t) => (
            <Select.Item<string>
              key={t ?? WHOLE_DRAWING}
              itemKey={t ?? WHOLE_DRAWING}
              size="small"
            >
              {targetName(parts, t)}
            </Select.Item>
          ))}
        </Select.Buttons>
      </Input.Item>
      {click.type === "fire" && (
        <Input.Item label="Command">
          <CommandButtons
            value={click.command}
            onChange={(command) => update({ ...click, command })}
            commands={commands}
          />
        </Input.Item>
      )}
    </Flex.Box>
  );
};
