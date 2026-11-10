// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/mqtt/task/EntryLabel.css";

import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { deep } from "@synnaxlabs/x";

import { type SparkplugTagID } from "@/feature/mqtt/task/types";
import { CSS } from "@/platform/css";
import { Task } from "@/platform/task";

interface PlainEntry {
  type: "plain";
  topic: string;
}

interface SparkplugEntry extends SparkplugTagID {
  type: "sparkplug";
}

export interface EntryLabelProps {
  /** The form path of a read entry or a write target. */
  path: deep.Target<PlainEntry | SparkplugEntry>;
}

const KINDS = {
  plain: { kind: "JSON", color: "var(--pluto-secondary-z)", placeholder: "New topic" },
  sparkplug: { kind: "SpB", color: "var(--pluto-primary-z)", placeholder: "New tag" },
} as const;

interface LabelProps {
  type: keyof typeof KINDS;
  identity: string;
}

const Label = ({ type, identity }: LabelProps) => {
  const { kind, color, placeholder } = KINDS[type];
  return (
    <Flex.Box x align="center" gap={1.5} className={CSS.B("mqtt-entry-label")}>
      <Task.Views.ItemLabel
        kind={kind}
        kindColor={color}
        color={identity === "" ? 8 : 10}
        variant="code"
      >
        {/* Isolated so the right-to-left overflow still reads left to right. */}
        {identity === "" ? placeholder : `⁦${identity}⁩`}
      </Task.Views.ItemLabel>
    </Flex.Box>
  );
};

const PlainLabel = ({ path }: { path: deep.Path<PlainEntry> }) => {
  const topic = PForm.useFieldValue(path.topic);
  return <Label type="plain" identity={topic} />;
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
  return <Label type="sparkplug" identity={identity} />;
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
