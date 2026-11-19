// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type kafka, status } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { compare, id, json, search, TimeSpan, TimeStamp, unique } from "@synnaxlabs/x";
import pluralize from "pluralize";
import { type ReactNode, useEffect, useState } from "react";

import { type Device } from "@/feature/kafka/device/types";
import {
  defaultPicks,
  freeKeys,
  type Picks,
  shapeKey,
} from "@/feature/kafka/task/additions";
import { browseTopics, sampleTopic } from "@/feature/kafka/task/scan";
import { type ReadRecord, type ReadTopic } from "@/feature/kafka/task/types";
import { CSS } from "@/platform/css";
import { ICD } from "@/platform/icd";
import { Modals } from "@/platform/modals";

interface Params {
  device: Device;
  /** The topic that the dialog shows first, or "" for the first that holds records. */
  topic: string;
  /** The topics that the task holds now. */
  topics: ReadTopic[];
  /** Names the Kafka key of a record, such as "Any key" for the empty key. */
  label: (recordKey: string) => string;
}

/** The fields picked from the shapes of a sample of one topic. */
export interface TopicPick {
  topic: string;
  shapes: json.Shape[];
  picks: Picks;
}

/** What the dialog adds: the fields picked from topics, or one record of a topic. */
export type Result =
  { variant: "picks"; topics: TopicPick[] } | { variant: "record"; topic: string };

const FRAME = "records";

const HINT =
  "Select the fields to read. Records that hold the same fields share a message.";

const INDEX: ICD.Picker.Lock = { checked: true, label: "Index" };

const FORMAT_NAMES: Record<kafka.Format, string> = {
  json: "JSON",
  schema_registry: "Schema Registry",
  text: "Text",
  binary: "Binary",
};

interface Listing extends kafka.BrowseResult {
  /** Names the browse, so each one opens a fresh tree and drops its samples. */
  key: string;
  /** When the browse answered. */
  listedAt: TimeStamp;
}

/** A topic of the list: one that the cluster lists, or one that it does not. */
interface Entry {
  name: string;
  /** What the browse says of the topic. Null for a topic that it did not list. */
  listed: kafka.Topic | null;
}

type Sample =
  | { variant: "loading" }
  | { variant: "success"; data: kafka.SampledTopic }
  | { variant: "error"; status: status.Status };

interface ListingSamples {
  listing: string | null;
  topics: Map<string, Sample>;
}

const NO_TOPICS: kafka.Topic[] = [];

const NO_RECORDS =
  "The topic holds no records. Add an empty record to set its fields by hand.";

const NO_JSON_RECORDS =
  "The topic holds no JSON records. Add an empty record to set its fields by hand.";

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

/** A partition that the browse could not measure can hide records. */
const holdsRecords = ({ records, error }: kafka.Topic): boolean =>
  records > 0 || error !== "";

const sampleable = ({ listed }: Entry): boolean =>
  listed == null || (listed.readable && holdsRecords(listed));

const formatRate = (rate: number): string =>
  `${rate >= 10 ? Math.round(rate) : rate.toFixed(1)}/s`;

const formatNewest = (newest: TimeStamp, listedAt: TimeStamp): string => {
  const span = listedAt.span(newest);
  if (span.lessThan(TimeSpan.SECOND)) return "newest just now";
  return `newest ${span.toString("semantic")} ago`;
};

/**
 * @returns the partitions, rate, format, and newest record of a topic, as of the time
 * that the browse answered.
 */
const describeTopic = (t: kafka.Topic, listedAt: TimeStamp): string =>
  [
    pluralize("partition", t.partitions, true),
    formatRate(t.rate),
    t.format == null ? null : FORMAT_NAMES[t.format],
    t.compacted ? "compacted" : null,
    t.newest.equals(TimeStamp.ZERO) ? null : formatNewest(t.newest, listedAt),
  ]
    .filter((part) => part != null)
    .join(" · ");

/** @returns the fields that a click on a topic picks: each but the index, by shape. */
const defaultsOf = (shapes: json.Shape[], records: ReadRecord[]): Picks =>
  new Map(
    shapes
      .filter((shape) => freeKeys(shape, records).length > 0)
      .map((shape): [string, Set<string>] => [shapeKey(shape), defaultPicks(shape)])
      .filter(([, pointers]) => pointers.size > 0),
  );

const countOf = (picks: Picks | undefined): number =>
  picks == null ? 0 : [...picks.values()].reduce((n, p) => n + p.size, 0);

/** @returns picks with the picks of topic replaced by value. */
const withTopic = (
  picks: Map<string, TopicPick>,
  topic: string,
  shapes: json.Shape[],
  value: Picks,
): Map<string, TopicPick> => {
  const next = new Map(picks);
  if (value.size === 0) next.delete(topic);
  else next.set(topic, { topic, shapes, picks: value });
  return next;
};

/**
 * @returns picks with the picks of topic moved onto the shapes of a new sample of it. A
 * pick of a shape that the new sample lacks drops.
 */
const rebase = (
  picks: Map<string, TopicPick>,
  topic: string,
  shapes: json.Shape[],
): Map<string, TopicPick> => {
  const pick = picks.get(topic);
  if (pick == null) return picks;
  const keys = new Set(shapes.map(shapeKey));
  const kept: Picks = new Map([...pick.picks].filter(([key]) => keys.has(key)));
  return withTopic(picks, topic, shapes, kept);
};

interface TopicValuesProps {
  name: string;
  /** What the browse says of the topic. */
  note: string;
  /** Says what the browse or the sample could not read, or null for nothing. */
  warning: string | null;
  sample: kafka.SampledTopic;
  records: ReadRecord[];
  label: Params["label"];
  value: Picks;
  onChange: (value: Picks) => void;
}

/** The fields of each shape of a sample of a topic, to pick. */
const TopicValues = ({
  name,
  note,
  warning,
  sample,
  records,
  label,
  value,
  onChange,
}: TopicValuesProps) => {
  const { shapes } = sample;
  // Shapes can share a pointer, so each field takes the index of its shape too.
  const fields = new Map<string, [shape: string, pointer: string]>();
  const groups = shapes.map((shape, i): ICD.Picker.ValueGroup => {
    const newest = newestFirst(sample, shape);
    const taken = freeKeys(shape, records).length === 0;
    const keys = shape.sources.map(label).join(", ");
    return {
      title: `${keys} · ${pluralize("record", shape.count, true)}`,
      values: shape.fields.map((f, j): ICD.Picker.Value => {
        const key = `${i}${f.pointer}`;
        fields.set(key, [shapeKey(shape), f.pointer]);
        let lock: ICD.Picker.Lock | undefined;
        if (j === shape.index) lock = INDEX;
        else if (taken) lock = ICD.Picker.TAKEN;
        return {
          key,
          name: f.pointer === "" ? "Whole value" : f.pointer,
          sample: show(newestValue(newest, f.pointer)),
          dataType: f.timeFormat == null ? f.dataType.toString() : "timestamp",
          lock,
        };
      }),
    };
  });
  const picked = shapes.flatMap((shape, i) =>
    [...(value.get(shapeKey(shape)) ?? [])].map((p) => `${i}${p}`),
  );
  const handleChange = (keys: string[]) => {
    const next: Picks = new Map();
    for (const key of keys) {
      const field = fields.get(key);
      if (field == null) throw new Error(`No field has the key ${key}`);
      const [shape, pointer] = field;
      next.set(shape, new Set([...(next.get(shape) ?? []), pointer]));
    }
    onChange(next);
  };
  const defaults = defaultsOf(shapes, records);
  return (
    <>
      <ICD.Picker.PaneHeader
        title={name}
        note={note}
        free={countOf(defaults)}
        picked={picked.length}
        onPickAll={() => onChange(defaults)}
        onClear={() => onChange(new Map())}
      />
      {warning != null && (
        <Text.Text
          level="small"
          status="warning"
          className={CSS.BE("kafka-add-data", "warning")}
        >
          {warning}
        </Text.Text>
      )}
      <ICD.Picker.Values
        noun="Field"
        groups={groups}
        value={picked}
        onChange={handleChange}
      />
    </>
  );
};

const pickedOf = (
  picked: number,
  free: number | null,
): ICD.Picker.TreeItem["picked"] => {
  if (picked === 0) return "none";
  return free != null && picked >= free ? "all" : "some";
};

const SKELETON: ICD.Picker.SkeletonItem[] = [
  [0, "14ch"],
  [0, "10ch"],
  [0, "16ch"],
  [0, "12ch"],
];

const Content = ({
  device,
  topic: initial,
  topics: held,
  label,
  close,
}: Modals.ContentProps<Params, Result>) => {
  const [term, setTerm] = useState("");
  const [selected, setSelected] = useState<string | null>(null);
  const [entered, setEntered] = useState<string[]>([]);
  const [picks, setPicks] = useState<Map<string, TopicPick>>(() => new Map());
  const [samples, setSamples] = useState<ListingSamples>(() => ({
    listing: null,
    topics: new Map(),
  }));
  const { result, listen, follow } = ICD.Picker.useListen<Listing>(
    async (client) => {
      const answer = await browseTopics(client, device);
      return { ...answer, key: id.create(), listedAt: TimeStamp.now() };
    },
    [device.key],
  );
  const listing = result.data;
  if (listing != null && samples.listing !== listing.key)
    setSamples({ listing: listing.key, topics: new Map() });

  const loadSample = async (topic: string): Promise<kafka.SampledTopic | null> => {
    const set = (sample: Sample) =>
      setSamples((prev) => ({
        ...prev,
        topics: new Map(prev.topics).set(topic, sample),
      }));
    set({ variant: "loading" });
    try {
      const data = await follow((client) => sampleTopic(client, device, topic));
      if (data == null) return null;
      set({ variant: "success", data });
      setPicks((prev) => rebase(prev, topic, data.shapes));
      return data;
    } catch (e) {
      set({
        variant: "error",
        status: status.fromException(e, {
          message: "Failed to read records",
          name: topic,
        }),
      });
      return null;
    }
  };

  const listed = listing?.topics ?? NO_TOPICS;
  const listedNames = new Set(listed.map(({ name }) => name));
  const unlisted = unique
    .unique([...held.map(({ name }) => name), ...entered])
    .filter((name) => name !== "" && !listedNames.has(name));
  const entries: Entry[] = [
    ...listed.map((t): Entry => ({ name: t.name, listed: t })),
    ...unlisted.map((name): Entry => ({ name, listed: null })),
  ].sort(compare.by(({ name }) => name));
  const recordsOf = (topic: string): ReadRecord[] =>
    held.find(({ name }) => name === topic)?.records ?? [];
  // A failed browse shows its error until the user selects a topic.
  let firstActive: string | null = null;
  if (listing != null)
    firstActive = entries.some(({ name }) => name === initial)
      ? initial
      : ((entries.find((e) => e.listed != null && sampleable(e)) ?? entries.at(0))
          ?.name ?? null);
  const active = selected ?? firstActive;
  const activeEntry = entries.find(({ name }) => name === active) ?? null;
  const activeSample = active == null ? undefined : samples.topics.get(active);
  useEffect(() => {
    if (activeEntry != null && sampleable(activeEntry) && activeSample == null)
      void loadSample(activeEntry.name);
  }, [activeEntry, activeSample, loadSample]);

  const trimmed = term.trim();
  const visible = entries.filter(({ name }) => search.matches(name, trimmed));
  const items = visible.map((entry): ICD.Picker.TreeItem => {
    const { name, listed: t } = entry;
    const sample = samples.topics.get(name);
    const picked = countOf(picks.get(name)?.picks);
    const free =
      sample?.variant === "success"
        ? countOf(defaultsOf(sample.data.shapes, recordsOf(name)))
        : null;
    let meta = t == null ? "Not listed" : formatRate(t.rate);
    if (picked > 0) meta = `${picked} selected`;
    else if (t != null && !t.readable) meta = "No access";
    else if (t != null && !holdsRecords(t)) meta = "Empty";
    else if (held.some((h) => h.name === name)) meta = "In task";
    const loading = sample?.variant === "loading";
    return {
      key: name,
      name,
      meta,
      picked: pickedOf(picked, free),
      disabled: !sampleable(entry) || free === 0 || loading,
      loading,
    };
  });

  const handlePick = async (topic: string) => {
    setSelected(topic);
    const sample = samples.topics.get(topic);
    const data = sample?.variant === "success" ? sample.data : await loadSample(topic);
    if (data == null) return;
    const defaults = defaultsOf(data.shapes, recordsOf(topic));
    setPicks((prev) => {
      const picked = countOf(prev.get(topic)?.picks);
      const all = countOf(defaults) > 0 && picked >= countOf(defaults);
      return withTopic(prev, topic, data.shapes, all ? new Map() : defaults);
    });
  };

  const handleEnter = () => {
    setEntered((prev) => unique.unique([...prev, trimmed]));
    setSelected(trimmed);
    setTerm("");
  };

  const browsing = result.variant === "loading";
  const browseFailed = result.variant === "error";
  const reading = entries.find(
    ({ name }) => samples.topics.get(name)?.variant === "loading",
  );
  let statusText = pluralize("topic", listed.length, true);
  if (browsing) statusText = "Listing topics";
  else if (reading != null) statusText = `Reading ${reading.name}`;
  else if (browseFailed) statusText = "Failed to list topics";
  else if (listing?.truncated === true) statusText += ", list is full";

  let emptyMessage = "The cluster has no topics.";
  if (trimmed !== "") emptyMessage = `No topic matches “${trimmed}”.`;
  else if (listing == null) emptyMessage = "Search for a topic to read it by name.";
  let tree: ReactNode;
  if (listing != null || browseFailed)
    tree = (
      <ICD.Picker.Tree
        key={listing?.key ?? "unlisted"}
        nodes={visible.map(({ name }) => ({ key: name }))}
        items={items}
        active={active}
        onActivate={setSelected}
        onPick={(topic) => void handlePick(topic)}
        emptyContent={<ICD.Picker.Notice message={emptyMessage} />}
      />
    );
  else tree = <ICD.Picker.TreeSkeleton items={SKELETON} />;

  const t = activeEntry?.listed ?? null;
  let pane: ReactNode;
  if (activeEntry == null)
    if (browseFailed) pane = <Status.Summary center status={result.status} />;
    else if (listing == null) pane = <ICD.Picker.PaneSkeleton noun="Field" />;
    else pane = <ICD.Picker.Notice message="Select a topic to see its records." />;
  else if (t != null && !t.readable)
    pane = (
      <ICD.Picker.Notice message="The cluster does not let Synnax read this topic." />
    );
  else if (t != null && !holdsRecords(t))
    pane = <ICD.Picker.Notice message={NO_RECORDS} />;
  else if (activeSample == null || activeSample.variant === "loading")
    pane = <ICD.Picker.PaneSkeleton noun="Field" title={activeEntry.name} />;
  else if (activeSample.variant === "error")
    pane = <Status.Summary center status={activeSample.status} />;
  else if (activeSample.data.shapes.length > 0) {
    const topic = activeEntry.name;
    const { data } = activeSample;
    const warning = [
      t == null || t.error === ""
        ? null
        : `Could not measure every partition. ${t.error}`,
      data.error === "" ? null : `Could not read every record. ${data.error}`,
    ]
      .filter((w) => w != null)
      .join(" ");
    pane = (
      <TopicValues
        key={`${samples.listing}/${topic}`}
        name={topic}
        note={
          t == null || listing == null
            ? "Not listed by the cluster"
            : describeTopic(t, listing.listedAt)
        }
        warning={warning === "" ? null : warning}
        sample={data}
        records={recordsOf(topic)}
        label={label}
        value={picks.get(topic)?.picks ?? new Map()}
        onChange={(value) =>
          setPicks((prev) => withTopic(prev, topic, data.shapes, value))
        }
      />
    );
  } else if (activeSample.data.error !== "")
    pane = (
      <Status.Summary
        center
        variant="warning"
        message="Read no records"
        description={activeSample.data.error}
      />
    );
  else pane = <ICD.Picker.Notice message={NO_JSON_RECORDS} />;

  const count = [...picks.values()].reduce((n, p) => n + countOf(p.picks), 0);
  const listedTerm = trimmed === "" || entries.some(({ name }) => name === trimmed);

  let sideAction: ReactNode = null;
  if (!listedTerm)
    sideAction = (
      <ICD.Picker.SideAction onClick={handleEnter}>
        {`Read “${trimmed}”`}
      </ICD.Picker.SideAction>
    );
  else if (activeEntry != null && t?.readable !== false)
    sideAction = (
      <ICD.Picker.SideAction
        onClick={() => close({ variant: "record", topic: activeEntry.name })}
      >
        Add an empty record
      </ICD.Picker.SideAction>
    );

  return (
    <ICD.Picker.Root value={FRAME} hint={HINT} className={CSS.B("kafka-add-data")}>
      <Modal.Header icon={<Icon.Logo.Kafka />}>Add data</Modal.Header>
      <ICD.Picker.Bar />
      <ICD.Picker.Frame itemKey={FRAME}>
        <ICD.Picker.Status
          status={statusText}
          loading={browsing || reading != null}
          onListen={listen}
        />
        <ICD.Picker.Side>
          <ICD.Picker.Search
            value={term}
            onChange={setTerm}
            placeholder="Search topics"
          />
          <ICD.Picker.SideList>{tree}</ICD.Picker.SideList>
          {sideAction}
        </ICD.Picker.Side>
        <ICD.Picker.Pane>{pane}</ICD.Picker.Pane>
      </ICD.Picker.Frame>
      <ICD.Picker.Footer
        counts={[["field", count]]}
        idle="field"
        onAdd={() =>
          close({
            variant: "picks",
            topics: [...picks.values()].sort(compare.by(({ topic }) => topic)),
          })
        }
        onCancel={() => close()}
      >
        {count > 0 && (
          <Text.Text level="small" color={10}>
            {`${pluralize("field", count, true)} selected`}
          </Text.Text>
        )}
      </ICD.Picker.Footer>
    </ICD.Picker.Root>
  );
};

/**
 * Opens a dialog that lists the topics of a cluster and of the task, and reads records
 * of the one selected, so the user can pick the fields of each shape of their values.
 * It resolves with the picks, or null when the user closes it.
 */
export const useAddDataModal = Modals.createPrompt<Result, Params>(Content);
