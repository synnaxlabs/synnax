// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { icd } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { ICD as PICD, Panel } from "@synnaxlabs/pluto";
import { array, type CrudeTimeSpan, deep, type payload, TimeSpan } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import { CSS } from "@/platform/css";
import { ICD } from "@/platform/icd";

const checkCycleTime = (span: TimeSpan): string | null => {
  if (span.lessThan(TimeSpan.NANOSECOND)) return "Cycle time must be 1ns or more";
  if (span.greaterThan(TimeSpan.MAX)) return "Cycle time is too long";
  return null;
};

interface CycleTimeProps {
  path: deep.Target<CrudeTimeSpan | undefined>;
}

/** Edits the cycle time of a message. A number typed alone is in milliseconds. */
const CycleTime = ({ path }: CycleTimeProps): ReactElement => {
  const { set, remove, mode } = Form.useContext();
  const state = Form.useFieldState(path, { optional: true });
  // The form store clones its values, which strips the TimeSpan class.
  const value = state?.value == null ? undefined : new TimeSpan(state.value);
  return (
    <Input.Item
      label="Cycle time"
      padHelpText
      helpText={state?.status.message}
      status={state?.status.variant}
    >
      <Input.TimeSpan
        inline
        unit={TimeSpan.MILLISECOND}
        precision={TimeSpan.NANOSECOND}
        value={value == null ? undefined : Number(value.valueOf())}
        onChange={(ns) => set(path, new TimeSpan(ns))}
        onClear={() => remove(path)}
        check={checkCycleTime}
        placeholder="None"
        aria-label="Cycle time"
        preview={mode === "preview"}
      />
    </Input.Item>
  );
};

export interface MessageProps {
  /** The form path of the message. */
  path: deep.Path<icd.Message>;
  /** Focuses the name and selects it, for a message the user just added. */
  autoFocus?: boolean;
}

const FOCUSED_NAME_INPUT_PROPS = { autoFocus: true, selectOnFocus: true } as const;

const FORMAT_NAMES: Record<payload.Format, string> = {
  binary: "Binary",
  text: "Text",
  json: "JSON",
};

const formatChangeDescription = (count: number): string =>
  count === 1
    ? "This removes its field. Tasks that bind it lose that channel."
    : `This removes its ${count} fields. Tasks that bind them lose those channels.`;

/**
 * Edits a message: its format, its cycle time, how frames match it, and its fields. A
 * format switch replaces the payload with an empty one of the new format. When the
 * message has fields, the switch asks first.
 */
export const Message = ({ path, autoFocus = false }: MessageProps): ReactElement => {
  const payloadPath = path.payload;
  const binary = deep.variant(payloadPath, "format", "binary");
  const text = deep.variant(payloadPath, "format", "text");
  const json = deep.variant(payloadPath, "format", "json");
  const { get, set, mode } = Form.useContext();
  const format = Form.useFieldValue(payloadPath.format);
  const { key } = Panel.useTabResource();
  const confirm = ICD.useConfirm();
  const handleFormatChange = async (next: payload.Format): Promise<void> => {
    const current = get<icd.Payload>(payloadPath, { optional: true })?.value;
    if (current == null || current.format === next) return;
    const { fields } = current;
    if (fields.length > 0) {
      const name = get<string>(path.name).value;
      const shown = name.trim() === "" ? "this message" : name;
      const confirmed = await confirm({
        icd: { key, name: get<string>(PICD.formPath.name).value },
        title: "ICD.Change format",
        message: `Change ${shown} to ${FORMAT_NAMES[next]}?`,
        description: formatChangeDescription(fields.length),
        confirm: { label: "Change format" },
      });
      if (confirmed !== true) return;
      // Another client can change the fields while the confirm is open. Ask again.
      const after = get<icd.Payload>(payloadPath, { optional: true })?.value.fields;
      const keys = (f: readonly { key: string }[] = []) => f.map(({ key }) => key);
      if (!array.equalAsSets(keys(after), keys(fields)))
        return await handleFormatChange(next);
    }
    set(payloadPath, ICD.createPayload(next));
  };
  return (
    <Form.Sections className={CSS.BE("icd-editor", "details")}>
      <Form.Section title="Message">
        <Flex.Box x>
          <Form.TextField
            path={path.name}
            label="Name"
            inputProps={autoFocus ? FOCUSED_NAME_INPUT_PROPS : undefined}
          />
          <Input.Item label="Format" padHelpText>
            <Select.Buttons<payload.Format>
              value={format}
              onChange={(next: payload.Format) => void handleFormatChange(next)}
              preview={mode === "preview"}
            >
              <Select.Item<payload.Format> itemKey="binary">Binary</Select.Item>
              <Select.Item<payload.Format> itemKey="text">Text</Select.Item>
              <Select.Item<payload.Format> itemKey="json">JSON</Select.Item>
            </Select.Buttons>
          </Input.Item>
        </Flex.Box>
        <CycleTime path={path.cycleTime} />
      </Form.Section>
      {format === "binary" && (
        <>
          <Form.Section title="Frame">
            <Flex.Box x>
              <ICD.Identifier path={binary} />
              <ICD.FormattedField<number>
                path={binary.length}
                label="Length"
                placeholder="Variable"
                unit="bytes"
                optional
                format={String}
                parse={(text) =>
                  ICD.orError(
                    ICD.parseUnsigned(text),
                    "Length must be a whole number of bytes",
                  )
                }
              />
            </Flex.Box>
            <ICD.IdentifierFields path={binary} />
          </Form.Section>
          <Form.Section title="Fields">
            <ICD.Fields path={binary.fields} encoding="binary" />
            <ICD.FieldsStatus path={binary.fields} />
          </Form.Section>
        </>
      )}
      {format === "text" && (
        <>
          <Form.Section title="Frame">
            <Flex.Box x>
              <Form.TextField path={text.prefix} label="Prefix" />
              <Form.TextField path={text.delimiter} label="Delimiter" />
            </Flex.Box>
          </Form.Section>
          <Form.Section title="Delimited fields">
            <ICD.Fields path={text.fields} encoding="delimited" />
          </Form.Section>
          <Form.Section title="Tagged fields">
            <ICD.Fields path={text.fields} encoding="tagged" />
            <ICD.FieldsStatus path={text.fields} />
          </Form.Section>
        </>
      )}
      {format === "json" && (
        <>
          <Form.Section title="Documents">
            <ICD.Match path={json} />
          </Form.Section>
          <Form.Section title="Fields">
            <ICD.Fields path={json.fields} encoding="json" />
            <ICD.FieldsStatus path={json.fields} />
          </Form.Section>
        </>
      )}
    </Form.Sections>
  );
};
