// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type kafka } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { useAsyncEffect } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Synnax as PSynnax } from "@synnaxlabs/pluto";
import { type json } from "@synnaxlabs/x";
import pluralize from "pluralize";
import { useState } from "react";

import { type Device } from "@/feature/kafka/device/types";
import { freeKeys, isPickable, type Picks } from "@/feature/kafka/task/additions";
import { sampleTopic } from "@/feature/kafka/task/sample";
import { type ReadRecord } from "@/feature/kafka/task/types";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";

interface Params {
  device: Device;
  topic: string;
  /** The records that the task holds now. */
  records: ReadRecord[];
  /** Names the Kafka key of a record, such as "Any key" for the empty key. */
  label: (recordKey: string) => string;
}

/** What the dialog adds: the fields picked from a sample, or one empty record. */
export type Result =
  { variant: "picks"; shapes: json.Shape[]; picks: Picks } | { variant: "record" };

type Sample =
  | { variant: "loading" }
  | { variant: "success"; data: kafka.SampledTopic }
  | { variant: "error"; error: unknown };

const unescapeToken = (token: string): string =>
  token.replaceAll("~1", "/").replaceAll("~0", "~");

/** @returns the value at a JSON pointer of a document, or undefined for none. */
const valueAt = (doc: unknown, pointer: string): unknown => {
  if (pointer === "") return doc;
  let value = doc;
  for (const token of pointer.slice(1).split("/").map(unescapeToken)) {
    if (value == null || typeof value !== "object") return undefined;
    value = (value as Record<string, unknown>)[token];
  }
  return value;
};

/** @returns the parsed value of the newest JSON record of a shape's keys. */
const newestValue = (sample: kafka.SampledTopic, shape: json.Shape): unknown => {
  const records = sample.records
    .filter(({ key, format }) => format === "json" && shape.sources.includes(key))
    .sort((a, b) => Number(b.time.valueOf() - a.time.valueOf()));
  try {
    return JSON.parse(records[0]?.payload ?? "null");
  } catch {
    // The payload holds only the start of a value longer than 16 KiB.
    return undefined;
  }
};

const show = (value: unknown): string => {
  if (value === undefined) return "–";
  return typeof value === "string" ? value : JSON.stringify(value);
};

interface FieldRowProps {
  field: json.InferredField;
  value: unknown;
  index: boolean;
  pickable: boolean;
  picked: boolean;
  onChange: (picked: boolean) => void;
}

const FieldRow = ({
  field,
  value,
  index,
  pickable,
  picked,
  onChange,
}: FieldRowProps) => {
  let type = field.timeFormat == null ? field.dataType.toString() : "timestamp";
  if (index) type = "index";
  return (
    <div className={CSS.BE("kafka-add-data", "field")}>
      <Input.Checkbox
        value={index || picked}
        onChange={onChange}
        disabled={index || !pickable}
        size="small"
        aria-label={`Read ${field.pointer}`}
      />
      <Text.Text level="small" weight={500} overflow="fade" variant="code">
        {field.pointer === "" ? "Whole value" : field.pointer}
      </Text.Text>
      <Text.Text level="small" color={9} overflow="fade" variant="code">
        {show(valueAt(value, field.pointer))}
      </Text.Text>
      <Text.Text level="small" color={9} overflow="fade">
        {type}
      </Text.Text>
    </div>
  );
};

interface ShapeProps {
  sample: kafka.SampledTopic;
  shape: json.Shape;
  records: ReadRecord[];
  label: Params["label"];
  picked: Set<string>;
  onChange: (picked: Set<string>) => void;
}

/** The fields of the records of one shape, as rows to pick. */
const Shape = ({ sample, shape, records, label, picked, onChange }: ShapeProps) => {
  const free = freeKeys(shape, records);
  const value = newestValue(sample, shape);
  const indexField = shape.index == null ? undefined : shape.fields[shape.index];
  const toggle = (pointer: string, on: boolean) => {
    const next = new Set(picked);
    if (on) next.add(pointer);
    else next.delete(pointer);
    onChange(next);
  };
  return (
    <Flex.Box y empty className={CSS.BE("kafka-add-data", "shape")}>
      <Text.Text
        level="small"
        weight={500}
        color={free.length === 0 ? 8 : 10}
        className={CSS.BE("kafka-add-data", "title")}
      >
        {shape.sources.map(label).join(", ")}
        <Text.Text el="span" level="small" color={9}>
          {free.length === 0 ? "In task" : pluralize("record", shape.count, true)}
        </Text.Text>
      </Text.Text>
      {shape.fields.map((field) => (
        <FieldRow
          key={field.pointer}
          field={field}
          value={value}
          index={field === indexField}
          pickable={free.length > 0 && isPickable(shape, picked, field)}
          picked={picked.has(field.pointer)}
          onChange={(on) => toggle(field.pointer, on)}
        />
      ))}
      {free.length > 0 && shape.fields.some((f) => !isPickable(shape, picked, f)) && (
        <Text.Text level="small" color={9} className={CSS.BE("kafka-add-data", "note")}>
          A string field cannot share a message with number, boolean, or timestamp
          fields.
        </Text.Text>
      )}
    </Flex.Box>
  );
};

const Content = ({
  device,
  topic,
  records,
  label,
  close,
}: Modals.ContentProps<Params, Result>) => {
  const client = PSynnax.use();
  const [sample, setSample] = useState<Sample>({ variant: "loading" });
  const [picks, setPicks] = useState<Picks>(() => new Map());
  useAsyncEffect(
    async (signal) => {
      if (client == null || topic === "") return;
      try {
        const data = await sampleTopic(client, device, topic);
        if (!signal.aborted) setSample({ variant: "success", data });
      } catch (error) {
        if (!signal.aborted) setSample({ variant: "error", error });
      }
    },
    [client, device.key, topic],
  );
  const shapes = sample.variant === "success" ? sample.data.shapes : [];
  const count = [...picks.values()].reduce((n, p) => n + p.size, 0);
  const handleAdd = () => close({ variant: "picks", shapes, picks });
  return (
    <Modal.Frame className={CSS.B("kafka-add-data")}>
      <Modal.Header icon={<Icon.Logo.Kafka />}>{["Add data", topic]}</Modal.Header>
      <Flex.Box y grow className={CSS.BE("kafka-add-data", "body")}>
        {topic === "" && (
          <Text.Text status="disabled">
            Set the topic of the task to read its records.
          </Text.Text>
        )}
        {topic !== "" && sample.variant === "loading" && (
          <Status.Summary variant="loading" message={`Reading records of ${topic}`} />
        )}
        {sample.variant === "error" && (
          <Status.Summary
            variant="error"
            message="Failed to read records"
            description={
              sample.error instanceof Error
                ? sample.error.message
                : String(sample.error)
            }
          />
        )}
        {sample.variant === "success" && sample.data.error !== "" && (
          <Status.Summary
            variant="warning"
            message="Read only some records"
            description={sample.data.error}
            center={false}
          />
        )}
        {sample.variant === "success" && shapes.length === 0 && (
          <Text.Text status="disabled">
            The topic holds no JSON records. Add a record to set its fields by hand.
          </Text.Text>
        )}
        {sample.variant === "success" &&
          shapes.map((shape, i) => (
            <Shape
              key={shape.sources.join("\n")}
              sample={sample.data}
              shape={shape}
              records={records}
              label={label}
              picked={picks.get(i) ?? new Set()}
              onChange={(p) => setPicks(new Map(picks).set(i, p))}
            />
          ))}
      </Flex.Box>
      <Modal.Footer>
        <Nav.Bar.Start>
          <Button.Button variant="text" onClick={() => close({ variant: "record" })}>
            <Icon.Add />
            Add empty record
          </Button.Button>
        </Nav.Bar.Start>
        <Nav.Bar.End x align="center" gap="small">
          <Button.Button variant="outlined" onClick={() => close()}>
            Cancel
          </Button.Button>
          <Button.Button variant="filled" disabled={count === 0} onClick={handleAdd}>
            {count === 0 ? "Add fields" : `Add ${pluralize("field", count, true)}`}
          </Button.Button>
        </Nav.Bar.End>
      </Modal.Footer>
    </Modal.Frame>
  );
};

/**
 * Opens a dialog that reads records of the task's topic, so the user can pick the
 * fields of each shape of their values. It resolves with the picks, or null when the
 * user closes it.
 */
export const useAddDataModal = Modals.createPrompt<Result, Params>(Content);
