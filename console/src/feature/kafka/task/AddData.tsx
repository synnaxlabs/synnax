// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type kafka } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { json } from "@synnaxlabs/x";
import pluralize from "pluralize";
import { type ReactNode, useState } from "react";

import { type Device } from "@/feature/kafka/device/types";
import {
  defaultPicks,
  freeKeys,
  type Picks,
  shapeKey,
} from "@/feature/kafka/task/additions";
import { sampleTopic } from "@/feature/kafka/task/sample";
import { type ReadRecord, type ReadTopic } from "@/feature/kafka/task/types";
import { CSS } from "@/platform/css";
import { ICD } from "@/platform/icd";
import { Modals } from "@/platform/modals";

interface Params {
  device: Device;
  /** The topic that the dialog reads first, or "" for none. */
  topic: string;
  /** The topics that the task holds now. */
  topics: ReadTopic[];
  /** Names the Kafka key of a record, such as "Any key" for the empty key. */
  label: (recordKey: string) => string;
}

/** What the dialog adds to a topic: the fields picked from a sample, or one record. */
export type Result = { topic: string } & (
  { variant: "picks"; shapes: json.Shape[]; picks: Picks } | { variant: "record" }
);

const FRAME = "records";

const INDEX: ICD.Picker.Lock = { checked: true, label: "Index" };

/** @returns the value at a JSON pointer of a document, or undefined for none. */
const valueAt = (doc: unknown, pointer: string): unknown => {
  let value = doc;
  for (const token of json.parsePointer(pointer)) {
    if (value == null || typeof value !== "object") return undefined;
    value = (value as Record<string, unknown>)[token];
  }
  return value;
};

const parse = (payload: string): unknown => {
  try {
    return JSON.parse(payload);
  } catch {
    // The payload holds only the start of a value longer than 16 KiB.
    return undefined;
  }
};

/** @returns the parsed JSON records of a shape's keys, newest first. */
const newestFirst = (sample: kafka.SampledTopic, shape: json.Shape): unknown[] =>
  sample.records
    .filter(({ key, format }) => format === "json" && shape.sources.includes(key))
    .sort((a, b) => Number(b.time.valueOf() - a.time.valueOf()))
    .map(({ payload }) => parse(payload));

/** @returns the value of a field in the newest record that holds it. */
const newestValue = (records: unknown[], pointer: string): unknown => {
  for (const record of records) {
    const value = valueAt(record, pointer);
    if (value !== undefined) return value;
  }
  return undefined;
};

const show = (value: unknown): string | null => {
  if (value === undefined) return null;
  return typeof value === "string" ? value : JSON.stringify(value);
};

interface ShapeValuesProps {
  sample: kafka.SampledTopic;
  shape: json.Shape;
  name: string;
  /** The task reads every key of the shape already. */
  taken: boolean;
  value: Set<string>;
  onChange: (value: Set<string>) => void;
}

/** The fields of one shape, to pick. */
const ShapeValues = ({
  sample,
  shape,
  name,
  taken,
  value,
  onChange,
}: ShapeValuesProps) => {
  const records = newestFirst(sample, shape);
  const values = shape.fields.map((f, i): ICD.Picker.Value => {
    const dataType = f.timeFormat == null ? f.dataType.toString() : "timestamp";
    let lock: ICD.Picker.Lock | undefined;
    if (i === shape.index) lock = INDEX;
    else if (taken) lock = ICD.Picker.TAKEN;
    return {
      key: f.pointer,
      name: f.pointer === "" ? "Whole value" : f.pointer,
      sample: show(newestValue(records, f.pointer)),
      dataType,
      lock,
    };
  });
  return (
    <>
      <ICD.Picker.PaneHeader
        title={name}
        note={pluralize("record", shape.count, true)}
        free={taken ? 0 : defaultPicks(shape).size}
        picked={value.size}
        onPickAll={() => onChange(defaultPicks(shape))}
        onClear={() => onChange(new Set())}
      />
      <ICD.Picker.Values
        noun="Field"
        groups={[{ values }]}
        value={[...value]}
        onChange={(keys) => onChange(new Set(keys))}
      />
    </>
  );
};

const toItem = (
  shape: json.Shape,
  picks: Picks,
  records: ReadRecord[],
  label: Params["label"],
): ICD.Picker.TreeItem => {
  const key = shapeKey(shape);
  const taken = freeKeys(shape, records).length === 0;
  const picked = picks.get(key)?.size ?? 0;
  const pickable = taken ? 0 : defaultPicks(shape).size;
  let meta = pluralize("record", shape.count, true);
  if (picked > 0) meta = `${picked} selected`;
  else if (taken) meta = "In task";
  let state: ICD.Picker.TreeItem["picked"] = "none";
  if (picked > 0) state = picked >= pickable ? "all" : "some";
  return {
    key,
    name: shape.sources.map(label).join(", "),
    meta,
    picked: state,
    disabled: pickable === 0,
  };
};

const SKELETON: ICD.Picker.SkeletonItem[] = [
  [0, "14ch"],
  [0, "10ch"],
  [0, "16ch"],
];

const Content = ({
  device,
  topic: initial,
  topics,
  label,
  close,
}: Modals.ContentProps<Params, Result>) => {
  const [topic, setTopic] = useState(initial);
  const records: ReadRecord[] = topics.find((t) => t.name === topic)?.records ?? [];
  const { result, listen } = ICD.Picker.useListen(
    async (client) => (topic === "" ? null : await sampleTopic(client, device, topic)),
    [device.key, topic],
  );
  const [picks, setPicks] = useState<Picks>(() => new Map());
  const [selected, setSelected] = useState<string | null>(null);
  // The listen keeps the last topic's sample while it reads a new topic.
  const sample = result.data?.topic === topic ? result.data : null;
  const shapes = sample?.shapes ?? [];
  const items = shapes.map((s) => toItem(s, picks, records, label));
  const active = shapes.find((s) => shapeKey(s) === selected) ?? shapes.at(0) ?? null;
  const loading = result.variant === "loading";
  const count = [...picks.values()].reduce((n, p) => n + p.size, 0);

  const setPicked = (shape: json.Shape, pointers: Set<string>) => {
    const next = new Map(picks);
    if (pointers.size === 0) next.delete(shapeKey(shape));
    else next.set(shapeKey(shape), pointers);
    setPicks(next);
  };

  const handlePick = (key: string) => {
    const shape = shapes.find((s) => shapeKey(s) === key);
    if (shape == null) throw new Error(`No shape of records has the key ${key}`);
    setPicked(shape, picks.has(key) ? new Set() : defaultPicks(shape));
    setSelected(key);
  };

  let list: ReactNode;
  if (result.variant === "error")
    list = (
      <Status.Summary
        center
        variant="error"
        message="Failed to read records"
        description={result.status.description}
      />
    );
  else if (topic === "")
    list = <ICD.Picker.Notice message="Enter a topic to read its records." />;
  else if (sample == null) list = <ICD.Picker.TreeSkeleton items={SKELETON} />;
  else if (shapes.length === 0 && sample.error !== "")
    list = (
      <Status.Summary
        center
        variant="warning"
        message="Read no records"
        description={sample.error}
      />
    );
  else
    list = (
      <ICD.Picker.Tree
        nodes={shapes.map((s) => ({ key: shapeKey(s) }))}
        items={items}
        active={active == null ? null : shapeKey(active)}
        onActivate={setSelected}
        onPick={handlePick}
        emptyContent={
          <ICD.Picker.Notice message="The topic holds no JSON records. Add a record to set its fields by hand." />
        }
      />
    );

  let pane: ReactNode;
  if (sample != null && active != null) {
    const key = shapeKey(active);
    pane = (
      <ShapeValues
        key={key}
        sample={sample}
        shape={active}
        name={active.sources.map(label).join(", ")}
        taken={freeKeys(active, records).length === 0}
        value={picks.get(key) ?? new Set()}
        onChange={(pointers) => setPicked(active, pointers)}
      />
    );
  } else if (sample == null && loading) pane = <ICD.Picker.PaneSkeleton noun="Field" />;
  else pane = <ICD.Picker.Notice message="Select a record key to see its fields." />;

  let status = pluralize("record", sample?.records.length ?? 0, true);
  if (loading) status = "Reading records";
  else if (sample != null && sample.error !== "" && shapes.length > 0)
    status += ", some unread";

  return (
    <ICD.Picker.Root value={FRAME} className={CSS.B("kafka-add-data")}>
      <Modal.Header icon={<Icon.Logo.Kafka />}>Add data</Modal.Header>
      <ICD.Picker.Bar>
        <Input.Text
          value={topic}
          onChange={(v) => {
            setTopic(v.trim());
            setPicks(new Map());
            setSelected(null);
          }}
          onlyChangeOnBlur
          autoFocus={initial === ""}
          size="small"
          aria-label="Topic"
          placeholder="Topic"
        />
      </ICD.Picker.Bar>
      <ICD.Picker.Frame itemKey={FRAME}>
        <ICD.Picker.Status status={status} loading={loading} onListen={listen} />
        <ICD.Picker.Side>
          <ICD.Picker.SideList>{list}</ICD.Picker.SideList>
          {topic !== "" && (
            <ICD.Picker.SideAction onClick={() => close({ topic, variant: "record" })}>
              Add an empty record
            </ICD.Picker.SideAction>
          )}
        </ICD.Picker.Side>
        <ICD.Picker.Pane>{pane}</ICD.Picker.Pane>
      </ICD.Picker.Frame>
      <ICD.Picker.Footer
        counts={[["field", count]]}
        idle="field"
        onAdd={() => close({ topic, variant: "picks", shapes, picks })}
        onCancel={() => close()}
      >
        <Text.Text level="small" color={count > 0 ? 10 : 9}>
          {count > 0
            ? `${pluralize("field", count, true)} selected`
            : "Select the fields to read. Records that hold the same fields share a message."}
        </Text.Text>
      </ICD.Picker.Footer>
    </ICD.Picker.Root>
  );
};

/**
 * Opens a dialog that reads records of a topic, so the user can pick the fields of each
 * shape of their values. It resolves with the topic and the picks, or null when the
 * user closes it.
 */
export const useAddDataModal = Modals.createPrompt<Result, Params>(Content);
