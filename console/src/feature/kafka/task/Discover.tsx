// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/kafka/task/Discover.css";

import { type icd, type kafka, status } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { compare, id, json, search, TimeSpan, TimeStamp, unique } from "@synnaxlabs/x";
import pluralize from "pluralize";
import { type ReactNode, useEffect, useState } from "react";

import type { Device } from "@/feature/kafka/device";
import {
  carriedFields,
  defaultPicks,
  isExtendable,
  type Picks,
  type Reads,
  shapeKey,
  takenPointers,
} from "@/feature/kafka/task/additions";
import { browseTopics, sampleTopic } from "@/feature/kafka/task/scan";
import { CSS } from "@/platform/css";
import { ICD } from "@/platform/icd";
import { Modals } from "@/platform/modals";

/** How the dialog picks topics, and the words for what a pick adds. */
export interface Kind {
  /** Names the dialog. */
  title: string;
  /** Says what a pick becomes, under the title of the pane. */
  hint: string;
  /** Names what one pick adds, such as "field". */
  noun: string;
  /** Starts the side action that adds a topic by name, such as "Read". */
  verb: string;
  /**
   * Each pick is a whole topic. It sends the shape of the most records of the topic, or
   * a message that a field under the topics picks.
   */
  whole?: boolean;
}

export interface Params {
  device: Device.Device;
  /** The topic that the dialog shows first, or "" for the first that holds records. */
  topic: string;
  /** The names of the topics that the task holds now. */
  topics: string[];
  /** What the task reads now from a topic, by its name. Absent, it reads nothing. */
  reads?: (topic: string) => Reads;
  /** Names the Kafka key of a record, such as "Any key" for the empty key. */
  label: (recordKey: string) => string;
}

/** The fields picked from the shapes of a sample of one topic. */
export interface TopicPick {
  topic: string;
  shapes: json.Shape[];
  picks: Picks;
}

/** A topic picked whole. */
export interface WholePick {
  topic: string;
  /** The shape of the most records of a sample of the topic, or null for none. */
  shape: json.Shape | null;
}

/** A message of an ICD. */
export interface Format {
  icd: icd.Key;
  message: icd.Message;
}

/** What the dialog adds. */
export interface Result {
  /** The fields picked from topics. */
  topics: TopicPick[];
  /** The topic that gets a record of any key with no fields, or null for none. */
  record: string | null;
  /** The topics picked whole. */
  whole: WholePick[];
  /** The message that each topic picked whole sends, or null for one from its shape. */
  format: Format | null;
}

const FRAME = "records";

const FORMAT_HELP = "Optional. A topic with no format gets one from its records.";

const NO_READS: Reads = { pointers: new Map(), jsonless: new Set(), anyKey: new Map() };

const readNothing = (): Reads => NO_READS;

const ANY_KEY: ICD.Picker.Lock = { checked: true, label: "Any key" };

const NOT_EXTENDABLE: ICD.Picker.Lock = { checked: false, label: "No JSON message" };

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

const UNREADABLE = "The cluster does not let Synnax read this topic.";

const NO_RECORDS = "The topic holds no records.";

const NO_JSON_RECORDS = "The topic holds no JSON records.";

const ADD_EMPTY = "Add an empty record to set its fields by hand.";

const NO_FIELDS = "Its target gets a message with no fields.";

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

/** @returns the fields that a click on a topic picks, by shape. */
const defaultsOf = (shapes: json.Shape[], reads: Reads): Picks =>
  new Map(
    shapes
      .map((shape): [string, Set<string>] => [
        shapeKey(shape),
        defaultPicks(shape, reads),
      ])
      .filter(([, pointers]) => pointers.size > 0),
  );

const countOf = (picks: Picks | undefined): number =>
  picks == null ? 0 : [...picks.values()].reduce((n, p) => n + p.size, 0);

/** The shape that each topic picked whole sends, by topic. */
type WholePicks = Map<string, json.Shape | null>;

/** @returns picks with topic picked whole to send shape, or dropped when not picked. */
const withWhole = (
  picks: WholePicks,
  topic: string,
  picked: boolean,
  shape: json.Shape | null,
): WholePicks => {
  const next = new Map(picks);
  if (picked) next.set(topic, shape);
  else next.delete(topic);
  return next;
};

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

/** @returns the value that a pane shows for a field of a shape. */
const valueOf = (
  key: string,
  field: json.InferredField,
  newest: unknown[],
): ICD.Picker.Value => ({
  key,
  name: field.pointer === "" ? "Whole value" : field.pointer,
  sample: show(newestValue(newest, field.pointer)),
  dataType: ICD.DEFAULT_DATA_TYPES[field.kind],
});

/** @returns the title of the fields of a shape: its keys and its record count. */
const titleOf = (shape: json.Shape, label: Params["label"]): string =>
  `${shape.sources.map(label).join(", ")} · ${pluralize("record", shape.count, true)}`;

const PaneWarning = ({ warning }: { warning: string | null }) =>
  warning == null ? null : (
    <Text.Text
      level="small"
      status="warning"
      className={CSS.BE("kafka-discover", "warning")}
    >
      {warning}
    </Text.Text>
  );

interface TopicValuesProps {
  name: string;
  /** What the browse says of the topic, when a browse answered. */
  note?: string;
  /** Says what the browse or the sample could not read, or null for nothing. */
  warning: string | null;
  sample: kafka.SampledTopic;
  reads: Reads;
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
  reads,
  label,
  value,
  onChange,
}: TopicValuesProps) => {
  const { shapes } = sample;
  // Shapes can share a pointer, so each field takes the index of its shape too.
  const fields = new Map<string, [shape: string, pointer: string]>();
  const groups = shapes.map((shape, i): ICD.Picker.ValueGroup => {
    const newest = newestFirst(sample, shape);
    const extendable = isExtendable(shape, reads);
    const taken = takenPointers(shape, reads);
    const carried = carriedFields(shape, reads);
    return {
      title: titleOf(shape, label),
      values: shape.fields.map((f, j): ICD.Picker.Value => {
        const key = `${i}${f.pointer}`;
        fields.set(key, [shapeKey(shape), f.pointer]);
        let lock: ICD.Picker.Lock | undefined;
        if (j === shape.index) lock = ICD.Picker.INDEX;
        else if (taken.has(f.pointer))
          lock = carried.has(f.pointer) ? ANY_KEY : ICD.Picker.TAKEN;
        else if (!extendable) lock = NOT_EXTENDABLE;
        return { ...valueOf(key, f, newest), lock };
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
  const defaults = defaultsOf(shapes, reads);
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
      <PaneWarning warning={warning} />
      <ICD.Picker.Values
        noun="Field"
        groups={groups}
        value={picked}
        onChange={handleChange}
      />
    </>
  );
};

interface WholeValuesProps {
  name: string;
  note?: string;
  warning: string | null;
  sample: kafka.SampledTopic;
  /** The shape that a pick of the topic sends. */
  shape: json.Shape;
  label: Params["label"];
  value: boolean;
  onChange: (value: boolean) => void;
}

/** The fields that a pick of a whole topic sends. */
const WholeValues = ({
  name,
  note,
  warning,
  sample,
  shape,
  label,
  value,
  onChange,
}: WholeValuesProps) => {
  const newest = newestFirst(sample, shape);
  const values = shape.fields.map((f) => valueOf(f.pointer, f, newest));
  const picked = value ? values.map(({ key }) => key) : [];
  return (
    <>
      <ICD.Picker.PaneHeader
        title={name}
        note={note}
        free={values.length}
        picked={picked.length}
        onPickAll={() => onChange(true)}
        onClear={() => onChange(false)}
      />
      <PaneWarning warning={warning} />
      <ICD.Picker.Values
        noun="Field"
        groups={[{ title: titleOf(shape, label), values }]}
        value={picked}
        // One tick or untick picks or drops the whole topic.
        onChange={(keys) => onChange(keys.length > picked.length)}
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

type ContentProps = Modals.ContentProps<Params, Result> & { kind: Kind };

const Content = ({
  device,
  topic: initial,
  topics: held,
  reads: readsOf = readNothing,
  label,
  close,
  kind,
}: ContentProps) => {
  const { whole = false } = kind;
  const [term, setTerm] = useState("");
  const [selected, setSelected] = useState<string | null>(null);
  const [entered, setEntered] = useState<string[]>([]);
  const [picks, setPicks] = useState<Map<string, TopicPick>>(() => new Map());
  const [wholePicks, setWholePicks] = useState<WholePicks>(() => new Map());
  const [format, setFormat] = useState<Format | null>(null);
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
      setWholePicks((prev) =>
        prev.has(topic)
          ? withWhole(
              prev,
              topic,
              true,
              ICD.Write.mostCommon(data.shapes, (shape) => shape),
            )
          : prev,
      );
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
    .unique([...held, ...entered, ...picks.keys(), ...wholePicks.keys()])
    .filter((name) => name !== "" && !listedNames.has(name));
  const entries: Entry[] = [
    ...listed.map((t): Entry => ({ name: t.name, listed: t })),
    ...unlisted.map((name): Entry => ({ name, listed: null })),
  ].sort(compare.by(({ name }) => name));
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
    const { name, listed: topic } = entry;
    const sample = samples.topics.get(name);
    let picked = countOf(picks.get(name)?.picks);
    let free =
      sample?.variant === "success"
        ? countOf(defaultsOf(sample.data.shapes, readsOf(name)))
        : null;
    if (whole) {
      picked = wholePicks.has(name) ? 1 : 0;
      free = 1;
    }
    let meta =
      topic != null ? formatRate(topic.rate) : listing == null ? "" : "Not listed";
    if (picked > 0) meta = `${picked} selected`;
    else if (!whole && topic != null && !topic.readable) meta = "No access";
    else if (topic != null && !holdsRecords(topic)) meta = "Empty";
    else if (held.includes(name)) meta = "In task";
    const loading = sample?.variant === "loading";
    return {
      key: name,
      name,
      meta,
      picked: pickedOf(picked, free),
      disabled: loading || (!whole && (!sampleable(entry) || free === 0)),
      loading,
    };
  });

  const pickWhole = async (topic: string) => {
    if (wholePicks.has(topic)) {
      setWholePicks((prev) => withWhole(prev, topic, false, null));
      return;
    }
    const sample = samples.topics.get(topic);
    const entry = entries.find(({ name }) => name === topic);
    let data = sample?.variant === "success" ? sample.data : null;
    if (data == null && (entry == null || sampleable(entry)))
      data = await loadSample(topic);
    const shape =
      data == null ? null : ICD.Write.mostCommon(data.shapes, (shape) => shape);
    setWholePicks((prev) => withWhole(prev, topic, true, shape));
  };

  const handlePick = async (topic: string) => {
    setSelected(topic);
    if (whole) {
      await pickWhole(topic);
      return;
    }
    const sample = samples.topics.get(topic);
    const data = sample?.variant === "success" ? sample.data : await loadSample(topic);
    if (data == null) return;
    const defaults = defaultsOf(data.shapes, readsOf(topic));
    setPicks((prev) => {
      const picked = countOf(prev.get(topic)?.picks);
      const all = countOf(defaults) > 0 && picked >= countOf(defaults);
      return withTopic(prev, topic, data.shapes, all ? new Map() : defaults);
    });
  };

  const handleTerm = () => {
    setEntered((prev) => unique.unique([...prev, trimmed]));
    setSelected(trimmed);
    setTerm("");
    if (whole) void pickWhole(trimmed);
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

  const activeTopic = activeEntry?.listed ?? null;
  const fill = whole ? NO_FIELDS : ADD_EMPTY;
  let pane: ReactNode;
  if (activeEntry == null && browseFailed)
    pane = <Status.Summary center status={result.status} />;
  else if (activeEntry == null && listing == null)
    pane = <ICD.Picker.PaneSkeleton noun="Field" />;
  else if (activeEntry == null)
    pane = <ICD.Picker.Notice message="Select a topic to see its records." />;
  else if (whole && format != null)
    pane = (
      <ICD.Picker.Notice
        message={`Each picked topic sends “${format.message.name}”.`}
      />
    );
  else if (activeTopic != null && !activeTopic.readable)
    pane = <ICD.Picker.Notice message={whole ? `${UNREADABLE} ${fill}` : UNREADABLE} />;
  else if (activeTopic != null && !holdsRecords(activeTopic))
    pane = <ICD.Picker.Notice message={`${NO_RECORDS} ${fill}`} />;
  else if (activeSample == null || activeSample.variant === "loading")
    pane = <ICD.Picker.PaneSkeleton noun="Field" title={activeEntry.name} />;
  else if (activeSample.variant === "error")
    pane = <Status.Summary center status={activeSample.status} />;
  else if (activeSample.data.shapes.length > 0) {
    const topic = activeEntry.name;
    const { data } = activeSample;
    const warning = [
      activeTopic == null || activeTopic.error === ""
        ? null
        : `Could not measure every partition. ${activeTopic.error}`,
      data.error === "" ? null : `Could not read every record. ${data.error}`,
    ]
      .filter((w) => w != null)
      .join(" ");
    const note =
      listing == null
        ? undefined
        : activeTopic == null
          ? "Not listed by the cluster"
          : describeTopic(activeTopic, listing.listedAt);
    const shape = ICD.Write.mostCommon(data.shapes, (shape) => shape);
    pane =
      whole && shape != null ? (
        <WholeValues
          key={`${samples.listing}/${topic}`}
          name={topic}
          note={note}
          warning={warning === "" ? null : warning}
          sample={data}
          shape={shape}
          label={label}
          value={wholePicks.has(topic)}
          onChange={(picked) =>
            setWholePicks((prev) => withWhole(prev, topic, picked, shape))
          }
        />
      ) : (
        <TopicValues
          key={`${samples.listing}/${topic}`}
          name={topic}
          note={note}
          warning={warning === "" ? null : warning}
          sample={data}
          reads={readsOf(topic)}
          label={label}
          value={picks.get(topic)?.picks ?? new Map()}
          onChange={(value) =>
            setPicks((prev) => withTopic(prev, topic, data.shapes, value))
          }
        />
      );
  } else if (
    whole &&
    listing != null &&
    activeTopic == null &&
    activeSample.data.records.length === 0
  )
    pane = (
      <Status.Summary
        center
        variant="info"
        message="Topic does not exist yet"
        description={`${NO_FIELDS} Add fields in the target pane.`}
      />
    );
  else if (activeSample.data.error !== "")
    pane = (
      <Status.Summary
        center
        variant="warning"
        message="Read no records"
        description={activeSample.data.error}
      />
    );
  else pane = <ICD.Picker.Notice message={`${NO_JSON_RECORDS} ${fill}`} />;

  const count = whole
    ? wholePicks.size
    : [...picks.values()].reduce((n, p) => n + countOf(p.picks), 0);
  const unknownTerm = trimmed !== "" && !entries.some(({ name }) => name === trimmed);
  const finish = (record: string | null) =>
    close({
      topics: [...picks.values()].sort(compare.by(({ topic }) => topic)),
      record,
      whole: [...wholePicks]
        .map(([topic, shape]): WholePick => ({ topic, shape }))
        .sort(compare.by(({ topic }) => topic)),
      format,
    });

  let sideAction: ReactNode = null;
  if (unknownTerm)
    sideAction = (
      <ICD.Picker.SideAction onClick={handleTerm}>
        {`${kind.verb} “${trimmed}”`}
      </ICD.Picker.SideAction>
    );
  else if (
    !whole &&
    activeEntry != null &&
    (activeTopic == null || activeTopic.readable)
  )
    sideAction = (
      <ICD.Picker.SideAction onClick={() => finish(activeEntry.name)}>
        Add an empty record
      </ICD.Picker.SideAction>
    );

  return (
    <ICD.Picker.Root value={FRAME} className={CSS.B("kafka-discover")}>
      <Modal.Header icon={<Icon.Logo.Kafka />}>{kind.title}</Modal.Header>
      <ICD.Picker.Bar />
      <ICD.Picker.Frame itemKey={FRAME} hint={kind.hint}>
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
          {whole && (
            <Flex.Box y empty className={CSS.BE("kafka-discover", "format")}>
              <ICD.MessageFormatField
                icdKey={format?.icd}
                message={format?.message.key}
                onChange={(icdKey, message) => setFormat({ icd: icdKey, message })}
                helpText={FORMAT_HELP}
                padHelpText
              />
            </Flex.Box>
          )}
        </ICD.Picker.Side>
        <ICD.Picker.Pane>{pane}</ICD.Picker.Pane>
      </ICD.Picker.Frame>
      <ICD.Picker.Footer
        counts={[[kind.noun, count]]}
        idle={kind.noun}
        onAdd={() => finish(null)}
        onCancel={() => close()}
      >
        {count > 0 && (
          <Text.Text level="small" color={10}>
            {`${pluralize(kind.noun, count, true)} selected`}
          </Text.Text>
        )}
      </ICD.Picker.Footer>
    </ICD.Picker.Root>
  );
};

/**
 * Creates a hook that opens a dialog of the topics of a cluster and of the task. The
 * user picks fields of the records of a topic, or whole topics, as kind says. The
 * dialog resolves with the picks, or null when the user closes it.
 */
export const createDiscoverModal = (kind: Kind): Modals.PromptHook<Params, Result> => {
  const Dialog = (props: Modals.ContentProps<Params, Result>) => (
    <Content {...props} kind={kind} />
  );
  return Modals.createPrompt<Result, Params>(Dialog);
};
