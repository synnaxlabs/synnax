// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/mqtt/task/EntryLabel.css";

import type { channel, icd } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import type { Tree } from "@synnaxlabs/lyra/tree";
import { ICD as PICD } from "@synnaxlabs/pluto";
import { deep, type payload } from "@synnaxlabs/x";

import type { SparkplugTagID } from "@/feature/mqtt/task/types";
import { CSS } from "@/platform/css";
import { ICD } from "@/platform/icd";
import { Task } from "@/platform/task";

interface MessageOf {
  message: icd.MessageKey;
}

/** A read entry names its messages, and a write target names its message. */
interface PlainEntry {
  type: "plain";
  topic: string;
  icd: icd.Key;
  messages?: MessageOf[];
  message?: MessageOf;
}

interface SparkplugEntry extends SparkplugTagID {
  type: "sparkplug";
}

export interface EntryLabelProps {
  /** The form path of a read entry or a write target. */
  path: deep.Target<PlainEntry | SparkplugEntry>;
}

const KINDS = {
  plain: { color: "var(--pluto-secondary-z)", placeholder: "New topic" },
  sparkplug: { color: "var(--pluto-primary-z)", placeholder: "New tag" },
} as const;

const FORMAT_KINDS: Record<payload.Format, string> = {
  json: "JSON",
  text: "TEXT",
  binary: "BINARY",
};

interface LabelProps {
  type: keyof typeof KINDS;
  kind: string;
  identity: string;
}

const Label = ({ type, kind, identity }: LabelProps) => {
  const { color, placeholder } = KINDS[type];
  return (
    <Flex.Box x align="center" gap={1.5} className={CSS.B("mqtt-entry-label")}>
      <Task.Views.ItemLabel
        kind={kind}
        kindColor={color}
        color={identity === "" ? 8 : 10}
        variant="code"
      >
        {identity === "" ? placeholder : identity}
      </Task.Views.ItemLabel>
    </Flex.Box>
  );
};

const PlainLabel = ({ path }: { path: deep.Path<PlainEntry> }) => {
  const formats = ICD.useFormats();
  const icdKey = PForm.useFieldValue(path.icd);
  const read = PForm.useFieldValue(deep.present(path.messages)[0].message, {
    optional: true,
  });
  const written = PForm.useFieldValue(deep.present(path.message).message, {
    optional: true,
  });
  // A topic with no message yet reads JSON.
  const format = PForm.useFieldValue(
    PICD.formPath.messages[read ?? written ?? ""].payload.format,
    { optional: true, ctx: formats.get(icdKey) },
  );
  return (
    <Label
      type="plain"
      kind={FORMAT_KINDS[format ?? "json"]}
      identity={PForm.useFieldValue(path.topic)}
    />
  );
};

const SparkplugLabel = ({ path }: { path: deep.Path<SparkplugEntry> }) => {
  const group = PForm.useFieldValue(path.group);
  const edgeNode = PForm.useFieldValue(path.edgeNode);
  const device = PForm.useFieldValue(path.device);
  const tag = PForm.useFieldValue(path.tag);
  const identity =
    tag === ""
      ? ""
      : [group, edgeNode, device, tag].filter((level) => level !== "").join("/");
  return <Label type="sparkplug" kind="SpB" identity={identity} />;
};

/**
 * Names an entry by its payload format and its identity: the topic of a plain entry,
 * or the full tag path of a Sparkplug B entry.
 */
export const EntryLabel = ({ path }: EntryLabelProps) => {
  const type = PForm.useFieldValue(deep.widen(path).type);
  return type === "plain" ? (
    <PlainLabel path={deep.variant(path, "type", "plain")} />
  ) : (
    <SparkplugLabel path={deep.variant(path, "type", "sparkplug")} />
  );
};

interface TreePlainEntry extends PlainEntry {
  disabled: boolean;
}

interface TreeSparkplugEntry extends SparkplugEntry {
  disabled: boolean;
  channel: channel.Key;
}

export interface EntryItemProps extends Tree.ItemRenderProps<string> {
  /** The form path of a read entry or a write target. */
  path: deep.Target<TreePlainEntry | TreeSparkplugEntry>;
}

/** An entry in the tree of a task, with its channel when it is a Sparkplug B tag. */
export const EntryItem = ({ path, ...props }: EntryItemProps) => {
  const wide = deep.widen(path);
  const type = PForm.useFieldValue(wide.type, { optional: true });
  const disabled = PForm.useFieldValue(wide.disabled, { optional: true }) === true;
  if (type == null) return null;
  return (
    <Task.Views.TreeGroupItem
      {...props}
      label={<EntryLabel path={path} />}
      channel={
        type === "sparkplug"
          ? deep.variant(path, "type", "sparkplug").channel
          : undefined
      }
      off={disabled}
    >
      <Task.EnabledCheckbox path={wide.disabled} />
    </Task.Views.TreeGroupItem>
  );
};
