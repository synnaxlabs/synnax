// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { z } from "zod";

import { MQTT } from "@/feature/mqtt";

const issuesOf = (result: z.ZodSafeParseResult<unknown>) =>
  (result.error?.issues ?? []).map(({ message, path }) => ({
    message,
    path: path.join("."),
  }));

describe("MQTT Task Types", () => {
  const field = { key: "f1", pointer: "/temperature", channel: 1 };
  const entry = { type: "plain", key: "e1", topic: "plant/temp", fields: [field] };
  const tagID = {
    group: "plant",
    edgeNode: "line1",
    device: "oven",
    tag: "temperature",
  };
  const tagEntry = { type: "sparkplug", key: "s1", ...tagID };
  const readConfig = (...entries: object[]) => ({ device: "dev-001", entries });

  describe("READ_SCHEMAS", () => {
    it("should validate the type literal", () => {
      expect(MQTT.Task.READ_SCHEMAS.type.parse(MQTT.Task.READ_TYPE)).toBe("mqtt_read");
    });

    it("should default a plain entry to QoS 0 that keeps retained messages", () => {
      const result = MQTT.Task.READ_SCHEMAS.config.parse(readConfig(entry));
      expect(result.entries[0]).toMatchObject({
        qos: "at_most_once",
        retainedIgnored: false,
        index: "",
        disabled: false,
      });
    });

    it("should accept a Sparkplug B entry in a stored config", () => {
      const result = MQTT.Task.READ_SCHEMAS.config.parse(
        readConfig({ type: "sparkplug", key: "s1", group: "g", edgeNode: "n" }),
      );
      expect(result.entries[0].type).toBe("sparkplug");
    });
  });

  describe("deployReadConfigZ", () => {
    it("should accept a plain entry with one field", () => {
      const result = MQTT.Task.deployReadConfigZ.safeParse(readConfig(entry));
      expect(issuesOf(result)).toEqual([]);
    });

    it("should require a device", () => {
      const result = MQTT.Task.deployReadConfigZ.safeParse({
        device: "",
        entries: [entry],
      });
      expect(issuesOf(result)).toEqual([
        { message: "Device is required", path: "device" },
      ]);
    });

    it("should reject an empty topic", () => {
      const result = MQTT.Task.deployReadConfigZ.safeParse(
        readConfig({ ...entry, topic: "" }),
      );
      expect(issuesOf(result)).toEqual([
        { message: "Topic is required", path: "entries.0.topic" },
      ]);
    });

    it.each(["plant/+/temp", "plant/#"])(
      "should reject the wildcard topic %s",
      (topic) => {
        const result = MQTT.Task.deployReadConfigZ.safeParse(
          readConfig({ ...entry, topic }),
        );
        expect(issuesOf(result)).toEqual([
          {
            message: "Topic must not hold the wildcards + or #",
            path: "entries.0.topic",
          },
        ]);
      },
    );

    it("should reject a topic that two enabled entries share", () => {
      const result = MQTT.Task.deployReadConfigZ.safeParse(
        readConfig(entry, {
          ...entry,
          key: "e2",
          fields: [{ ...field, key: "f2", channel: 2 }],
        }),
      );
      expect(issuesOf(result)).toEqual([
        { message: 'Duplicate topic "plant/temp"', path: "entries.1.topic" },
      ]);
    });

    it("should allow a disabled entry to share a topic with an enabled one", () => {
      const result = MQTT.Task.deployReadConfigZ.safeParse(
        readConfig(entry, { ...entry, key: "e2", disabled: true }),
      );
      expect(issuesOf(result)).toEqual([]);
    });

    it("should reject a config with no entries", () => {
      const result = MQTT.Task.deployReadConfigZ.safeParse(readConfig());
      expect(issuesOf(result)).toEqual([
        { message: "At least one entry must be enabled", path: "entries" },
      ]);
    });

    it("should reject a config whose entries are all disabled", () => {
      const result = MQTT.Task.deployReadConfigZ.safeParse(
        readConfig({ ...entry, disabled: true }),
      );
      expect(issuesOf(result)).toEqual([
        { message: "At least one entry must be enabled", path: "entries" },
      ]);
    });

    it("should accept a Sparkplug B entry beside a plain entry", () => {
      const result = MQTT.Task.deployReadConfigZ.safeParse(readConfig(entry, tagEntry));
      expect(issuesOf(result)).toEqual([]);
    });

    it("should accept a Sparkplug B entry with a timestamp data type", () => {
      const result = MQTT.Task.deployReadConfigZ.safeParse(
        readConfig({ ...tagEntry, dataType: "timestamp" }),
      );
      expect(issuesOf(result)).toEqual([]);
    });

    it("should require the group, the edge node, and the tag of a Sparkplug B entry", () => {
      const result = MQTT.Task.deployReadConfigZ.safeParse(
        readConfig({ ...tagEntry, group: "", edgeNode: "", tag: "" }),
      );
      expect(issuesOf(result)).toEqual([
        { message: "Group is required", path: "entries.0.group" },
        { message: "Edge node is required", path: "entries.0.edgeNode" },
        { message: "Tag is required", path: "entries.0.tag" },
      ]);
    });

    it.each([
      ["group", "Group"],
      ["edgeNode", "Edge node"],
      ["device", "Device"],
    ])("should reject a %s that holds a topic character", (field, label) => {
      for (const id of ["a/b", "a+", "#"]) {
        const result = MQTT.Task.deployReadConfigZ.safeParse(
          readConfig({ ...tagEntry, [field]: id }),
        );
        expect(issuesOf(result)).toEqual([
          { message: `${label} must not hold /, +, or #`, path: `entries.0.${field}` },
        ]);
      }
    });

    it("should accept a tag name that holds a slash", () => {
      const result = MQTT.Task.deployReadConfigZ.safeParse(
        readConfig({ ...tagEntry, tag: "Node Control/Rebirth" }),
      );
      expect(issuesOf(result)).toEqual([]);
    });

    it("should reject a tag that two enabled entries share", () => {
      const result = MQTT.Task.deployReadConfigZ.safeParse(
        readConfig(tagEntry, { ...tagEntry, key: "s2" }),
      );
      expect(issuesOf(result)).toEqual([
        { message: 'Duplicate tag "temperature"', path: "entries.1.tag" },
      ]);
    });

    it("should allow a disabled entry to share a tag with an enabled one", () => {
      const result = MQTT.Task.deployReadConfigZ.safeParse(
        readConfig(tagEntry, { ...tagEntry, key: "s2", disabled: true }),
      );
      expect(issuesOf(result)).toEqual([]);
    });

    it("should tell a tag of a device from a tag of the edge node", () => {
      const result = MQTT.Task.deployReadConfigZ.safeParse(
        readConfig(
          { ...tagEntry, device: "", tag: "oven/temperature" },
          { ...tagEntry, key: "s2", device: "oven", tag: "temperature" },
        ),
      );
      expect(issuesOf(result)).toEqual([]);
    });

    it("should allow a plain topic that reads like the name of a tag", () => {
      const result = MQTT.Task.deployReadConfigZ.safeParse(
        readConfig({ ...entry, topic: "temperature" }, tagEntry),
      );
      expect(issuesOf(result)).toEqual([]);
    });

    it("should count a Sparkplug B entry as an enabled entry", () => {
      const result = MQTT.Task.deployReadConfigZ.safeParse(
        readConfig({ ...entry, disabled: true }, tagEntry),
      );
      expect(issuesOf(result)).toEqual([]);
    });

    it("should reject a pointer that two fields of an entry share", () => {
      const result = MQTT.Task.deployReadConfigZ.safeParse(
        readConfig({
          ...entry,
          fields: [field, { ...field, key: "f2", channel: 2 }],
        }),
      );
      expect(issuesOf(result)).toEqual([
        {
          message: 'Pointer "/temperature" is already used by another field',
          path: "entries.0.fields.1.pointer",
        },
      ]);
    });

    it("should reject two fields that take the whole payload", () => {
      const result = MQTT.Task.deployReadConfigZ.safeParse(
        readConfig({
          ...entry,
          fields: [
            { ...field, pointer: "" },
            { ...field, key: "f2", channel: 2, pointer: "" },
          ],
        }),
      );
      expect(issuesOf(result)).toEqual([
        {
          message: 'Pointer "" is already used by another field',
          path: "entries.0.fields.1.pointer",
        },
      ]);
    });

    it("should allow the same pointer in two entries", () => {
      const result = MQTT.Task.deployReadConfigZ.safeParse(
        readConfig(entry, {
          ...entry,
          key: "e2",
          topic: "plant/other",
          fields: [{ ...field, key: "f2", channel: 2 }],
        }),
      );
      expect(issuesOf(result)).toEqual([]);
    });

    it("should reject a pointer that is not RFC 6901", () => {
      const result = MQTT.Task.deployReadConfigZ.safeParse(
        readConfig({ ...entry, fields: [{ ...field, pointer: "temperature" }] }),
      );
      expect(issuesOf(result)).toEqual([
        {
          message: "must be a valid JSON pointer (RFC 6901)",
          path: "entries.0.fields.0.pointer",
        },
      ]);
    });

    it("should reject an index field with no time format", () => {
      const result = MQTT.Task.deployReadConfigZ.safeParse(
        readConfig({
          ...entry,
          index: "ts",
          fields: [field, { key: "ts", pointer: "/ts", channel: 2 }],
        }),
      );
      expect(issuesOf(result)).toEqual([
        {
          message: "A timestamp field requires a time format",
          path: "entries.0.fields.1.timeFormat",
        },
      ]);
    });

    it("should reject a timestamp field with no time format", () => {
      const result = MQTT.Task.deployReadConfigZ.safeParse(
        readConfig({ ...entry, fields: [{ ...field, dataType: "timestamp" }] }),
      );
      expect(issuesOf(result)).toEqual([
        {
          message: "A timestamp field requires a time format",
          path: "entries.0.fields.0.timeFormat",
        },
      ]);
    });

    it("should accept an index field with a time format", () => {
      const result = MQTT.Task.deployReadConfigZ.safeParse(
        readConfig({
          ...entry,
          index: "ts",
          fields: [
            field,
            { key: "ts", pointer: "/ts", channel: 2, timeFormat: "unix_ms" },
          ],
        }),
      );
      expect(issuesOf(result)).toEqual([]);
    });

    it("should reject fixed-length and variable-length fields in one entry", () => {
      const result = MQTT.Task.deployReadConfigZ.safeParse(
        readConfig({
          ...entry,
          fields: [
            field,
            { key: "f2", pointer: "/unit", channel: 2, dataType: "string" },
          ],
        }),
      );
      expect(issuesOf(result)).toEqual([
        {
          message:
            "The fields of a topic share one index, so a variable-length data type cannot mix with the others",
          path: "entries.0.fields.1.dataType",
        },
      ]);
    });

    it("should allow a disabled variable-length field beside fixed-length fields", () => {
      const result = MQTT.Task.deployReadConfigZ.safeParse(
        readConfig({
          ...entry,
          fields: [
            field,
            {
              key: "f2",
              pointer: "/unit",
              channel: 2,
              dataType: "string",
              disabled: true,
            },
          ],
        }),
      );
      expect(issuesOf(result)).toEqual([]);
    });

    it("should reject an enum label that a field maps twice", () => {
      const result = MQTT.Task.deployReadConfigZ.safeParse(
        readConfig({
          ...entry,
          fields: [
            {
              ...field,
              enumValues: [
                { label: "ON", value: 1 },
                { label: "ON", value: 2 },
              ],
            },
          ],
        }),
      );
      expect(issuesOf(result)).toEqual([
        {
          message: 'Duplicate enum label "ON"',
          path: "entries.0.fields.0.enumValues.1.label",
        },
      ]);
    });

    it("should reject a channel that two fields of an entry write", () => {
      const result = MQTT.Task.deployReadConfigZ.safeParse(
        readConfig({
          ...entry,
          fields: [field, { ...field, key: "f2", pointer: "/pressure" }],
        }),
      );
      expect(issuesOf(result).map(({ path }) => path)).toEqual([
        "entries.0.fields.0.channel",
        "entries.0.fields.1.channel",
      ]);
    });
  });

  const target = {
    type: "plain",
    key: "t1",
    topic: "plant/valve/set",
    channel: { pointer: "/value", channel: 1 },
  };
  const tagTarget = { type: "sparkplug", key: "s1", ...tagID };
  const writeConfig = (...targets: object[]) => ({ device: "dev-001", targets });

  describe("WRITE_SCHEMAS", () => {
    it("should validate the type literal", () => {
      expect(MQTT.Task.WRITE_SCHEMAS.type.parse(MQTT.Task.WRITE_TYPE)).toBe(
        "mqtt_write",
      );
    });

    it("should default a plain target to QoS 1 with no retain", () => {
      const result = MQTT.Task.WRITE_SCHEMAS.config.parse(writeConfig(target));
      expect(result.targets[0]).toMatchObject({
        qos: "at_least_once",
        retained: false,
        fields: [],
      });
    });
  });

  describe("deployWriteConfigZ", () => {
    it("should accept a plain target", () => {
      const result = MQTT.Task.deployWriteConfigZ.safeParse(writeConfig(target));
      expect(issuesOf(result)).toEqual([]);
    });

    it("should reject an empty topic", () => {
      const result = MQTT.Task.deployWriteConfigZ.safeParse(
        writeConfig({ ...target, topic: "" }),
      );
      expect(issuesOf(result)).toEqual([
        { message: "Topic is required", path: "targets.0.topic" },
      ]);
    });

    it("should reject a wildcard topic", () => {
      const result = MQTT.Task.deployWriteConfigZ.safeParse(
        writeConfig({ ...target, topic: "plant/+/set" }),
      );
      expect(issuesOf(result)).toEqual([
        {
          message: "Topic must not hold the wildcards + or #",
          path: "targets.0.topic",
        },
      ]);
    });

    it("should reject a config whose targets are all disabled", () => {
      const result = MQTT.Task.deployWriteConfigZ.safeParse(
        writeConfig({ ...target, disabled: true }),
      );
      expect(issuesOf(result)).toEqual([
        { message: "At least one target must be enabled", path: "targets" },
      ]);
    });

    it("should accept a Sparkplug B target beside a plain target", () => {
      const result = MQTT.Task.deployWriteConfigZ.safeParse(
        writeConfig(target, tagTarget),
      );
      expect(issuesOf(result)).toEqual([]);
    });

    it("should default a Sparkplug B target to the double type", () => {
      const result = MQTT.Task.deployWriteConfigZ.parse(writeConfig(tagTarget));
      expect(result.targets[0]).toMatchObject({ sparkplugType: "double", channel: 0 });
    });

    it("should reject a Sparkplug B type that the driver does not send", () => {
      const result = MQTT.Task.deployWriteConfigZ.safeParse(
        writeConfig({ ...tagTarget, sparkplugType: "Text" }),
      );
      expect(issuesOf(result).map(({ path }) => path)).toEqual([
        "targets.0.sparkplugType",
      ]);
    });

    it("should require the group, the edge node, and the tag of a Sparkplug B target", () => {
      const result = MQTT.Task.deployWriteConfigZ.safeParse(
        writeConfig({ ...tagTarget, group: "", edgeNode: "", tag: "" }),
      );
      expect(issuesOf(result)).toEqual([
        { message: "Group is required", path: "targets.0.group" },
        { message: "Edge node is required", path: "targets.0.edgeNode" },
        { message: "Tag is required", path: "targets.0.tag" },
      ]);
    });

    it("should reject a device ID that holds a topic character", () => {
      const result = MQTT.Task.deployWriteConfigZ.safeParse(
        writeConfig({ ...tagTarget, device: "oven/1" }),
      );
      expect(issuesOf(result)).toEqual([
        { message: "Device must not hold /, +, or #", path: "targets.0.device" },
      ]);
    });

    it("should reject a tag that two enabled targets share", () => {
      const result = MQTT.Task.deployWriteConfigZ.safeParse(
        writeConfig(tagTarget, { ...tagTarget, key: "s2" }),
      );
      expect(issuesOf(result)).toEqual([
        { message: 'Duplicate tag "temperature"', path: "targets.1.tag" },
      ]);
    });

    it("should reject additional fields when the channel value is the whole payload", () => {
      const result = MQTT.Task.deployWriteConfigZ.safeParse(
        writeConfig({
          ...target,
          channel: { pointer: "", channel: 1 },
          fields: [
            {
              key: "s1",
              type: "static",
              pointer: "/unit",
              jsonType: "string",
              value: "C",
            },
          ],
        }),
      );
      expect(issuesOf(result)).toEqual([
        {
          message:
            "An empty channel pointer sends the raw value as the payload, so additional fields are not allowed",
          path: "targets.0.channel.pointer",
        },
      ]);
    });

    it("should allow the channel value as the whole payload with no other fields", () => {
      const result = MQTT.Task.deployWriteConfigZ.safeParse(
        writeConfig({ ...target, channel: { pointer: "", channel: 1 } }),
      );
      expect(issuesOf(result)).toEqual([]);
    });

    it("should reject an additional field with no pointer", () => {
      const result = MQTT.Task.deployWriteConfigZ.safeParse(
        writeConfig({
          ...target,
          fields: [{ key: "g1", type: "generated", pointer: "", generator: "uuid" }],
        }),
      );
      expect(issuesOf(result)).toEqual([
        {
          message: "Additional field pointer cannot be empty",
          path: "targets.0.fields.0.pointer",
        },
      ]);
    });

    it("should reject an additional field at the pointer of the channel", () => {
      const result = MQTT.Task.deployWriteConfigZ.safeParse(
        writeConfig({
          ...target,
          fields: [
            { key: "g1", type: "generated", pointer: "/value", generator: "uuid" },
          ],
        }),
      );
      expect(issuesOf(result)).toEqual([
        {
          message: 'Pointer "/value" is already used by another field',
          path: "targets.0.fields.0.pointer",
        },
      ]);
    });

    it("should reject enum values on a JSON type other than string", () => {
      const result = MQTT.Task.deployWriteConfigZ.safeParse(
        writeConfig({
          ...target,
          channel: {
            ...target.channel,
            jsonType: "number",
            enumValues: [{ label: "ON", value: 1 }],
          },
        }),
      );
      expect(issuesOf(result)).toEqual([
        {
          message: "Enum values require the string JSON type",
          path: "targets.0.channel.jsonType",
        },
      ]);
    });

    it("should reject a channel value that two enum labels map", () => {
      const result = MQTT.Task.deployWriteConfigZ.safeParse(
        writeConfig({
          ...target,
          channel: {
            ...target.channel,
            jsonType: "string",
            enumValues: [
              { label: "ON", value: 1 },
              { label: "OPEN", value: 1 },
            ],
          },
        }),
      );
      expect(issuesOf(result)).toEqual([
        {
          message: "Duplicate enum value 1",
          path: "targets.0.channel.enumValues.1.value",
        },
      ]);
    });

    it("should reject a timestamp channel with no time format", () => {
      const result = MQTT.Task.deployWriteConfigZ.safeParse(
        writeConfig({
          ...target,
          channel: { ...target.channel, dataType: "timestamp" },
        }),
      );
      expect(issuesOf(result)).toEqual([
        {
          message: "A timestamp channel requires a time format",
          path: "targets.0.channel.timeFormat",
        },
      ]);
    });

    it("should reject a static value that is not a JSON primitive", () => {
      const result = MQTT.Task.deployWriteConfigZ.safeParse(
        writeConfig({
          ...target,
          fields: [
            {
              key: "s1",
              type: "static",
              pointer: "/meta",
              jsonType: "string",
              value: { nested: true },
            },
          ],
        }),
      );
      expect(issuesOf(result).map(({ path }) => path)).toEqual([
        "targets.0.fields.0.value",
      ]);
    });
  });
});

describe("MQTT Edge Task", () => {
  const tag = { key: "t1", name: "oven/temperature", channel: 1 };
  const edgeConfig = (...tags: object[]) => ({
    device: "dev-001",
    group: "plant",
    edgeNode: "line1",
    tags,
  });

  describe("EDGE_SCHEMAS", () => {
    it("should validate the type literal", () => {
      expect(MQTT.Task.EDGE_SCHEMAS.type.parse(MQTT.Task.EDGE_TYPE)).toBe(
        "mqtt_sparkplug_edge",
      );
    });

    it("should default a tag to the double type with no command channel", () => {
      const result = MQTT.Task.EDGE_SCHEMAS.config.parse(edgeConfig({ key: "t1" }));
      expect(result.authority).toBe(0);
      expect(result.tags[0]).toMatchObject({
        name: "",
        channel: 0,
        sparkplugType: "double",
        commandChannel: 0,
        disabled: false,
      });
    });
  });

  describe("deployEdgeConfigZ", () => {
    it("should accept a tag bound to a channel", () => {
      const result = MQTT.Task.deployEdgeConfigZ.safeParse(edgeConfig(tag));
      expect(issuesOf(result)).toEqual([]);
    });

    it("should require a device", () => {
      const result = MQTT.Task.deployEdgeConfigZ.safeParse({
        ...edgeConfig(tag),
        device: "",
      });
      expect(issuesOf(result)).toEqual([
        { message: "Device is required", path: "device" },
      ]);
    });

    it("should require the group and the edge node", () => {
      const result = MQTT.Task.deployEdgeConfigZ.safeParse({
        ...edgeConfig(tag),
        group: "",
        edgeNode: "",
      });
      expect(issuesOf(result)).toEqual([
        { message: "Group is required", path: "group" },
        { message: "Edge node is required", path: "edgeNode" },
      ]);
    });

    it.each([
      ["group", "Group"],
      ["edgeNode", "Edge node"],
    ])("should reject a %s that holds a topic character", (field, label) => {
      for (const id of ["a/b", "a+", "#"]) {
        const result = MQTT.Task.deployEdgeConfigZ.safeParse({
          ...edgeConfig(tag),
          [field]: id,
        });
        expect(issuesOf(result)).toEqual([
          { message: `${label} must not hold /, +, or #`, path: field },
        ]);
      }
    });

    it.each([-1, 256])("should reject the authority %i", (authority) => {
      const result = MQTT.Task.deployEdgeConfigZ.safeParse({
        ...edgeConfig(tag),
        authority,
      });
      expect(issuesOf(result).map(({ path }) => path)).toEqual(["authority"]);
    });

    it("should reject a config with no tags", () => {
      const result = MQTT.Task.deployEdgeConfigZ.safeParse(edgeConfig());
      expect(issuesOf(result)).toEqual([
        { message: "At least one tag must be enabled", path: "tags" },
      ]);
    });

    it("should reject a config whose tags are all disabled", () => {
      const result = MQTT.Task.deployEdgeConfigZ.safeParse(
        edgeConfig({ ...tag, disabled: true }),
      );
      expect(issuesOf(result)).toEqual([
        { message: "At least one tag must be enabled", path: "tags" },
      ]);
    });

    it("should require the name and the channel of a tag", () => {
      const result = MQTT.Task.deployEdgeConfigZ.safeParse(
        edgeConfig({ key: "t1", name: "", channel: 0 }),
      );
      expect(issuesOf(result)).toEqual([
        { message: "Name is required", path: "tags.0.name" },
        { message: "A channel is required", path: "tags.0.channel" },
      ]);
    });

    it("should reject a name that two enabled tags share", () => {
      const result = MQTT.Task.deployEdgeConfigZ.safeParse(
        edgeConfig(tag, { ...tag, key: "t2", channel: 2 }),
      );
      expect(issuesOf(result)).toEqual([
        { message: 'Duplicate name "oven/temperature"', path: "tags.1.name" },
      ]);
    });

    it("should allow a disabled tag to share a name with an enabled one", () => {
      const result = MQTT.Task.deployEdgeConfigZ.safeParse(
        edgeConfig(tag, { ...tag, key: "t2", disabled: true }),
      );
      expect(issuesOf(result)).toEqual([]);
    });

    it("should allow two enabled tags to publish one channel", () => {
      const result = MQTT.Task.deployEdgeConfigZ.safeParse(
        edgeConfig(tag, { ...tag, key: "t2", name: "oven/temperature_copy" }),
      );
      expect(issuesOf(result)).toEqual([]);
    });

    it("should reject a Sparkplug B type that the driver does not send", () => {
      const result = MQTT.Task.deployEdgeConfigZ.safeParse(
        edgeConfig({ ...tag, sparkplugType: "Text" }),
      );
      expect(issuesOf(result).map(({ path }) => path)).toEqual([
        "tags.0.sparkplugType",
      ]);
    });
  });
});

describe("MQTT Scan Task", () => {
  it("should accept a null and an absent config", () => {
    expect(z.validate(MQTT.Task.SCAN_SCHEMAS.config, null)).toBe(true);
    expect(z.validate(MQTT.Task.SCAN_SCHEMAS.config, undefined)).toBe(true);
  });

  it("should accept the null status data of a test connection reply", () => {
    expect(z.validate(MQTT.Task.SCAN_SCHEMAS.statusData, null)).toBe(true);
  });

  it("should parse the status data of a browse reply", () => {
    const data = {
      topics: [{ topic: "plant/temp", payload: "21.5", retained: true }],
      truncated: false,
    };
    expect(MQTT.Task.SCAN_SCHEMAS.statusData.parse(data)).toEqual(data);
  });

  it("should parse the edge nodes of a Sparkplug B browse reply", () => {
    const data = {
      nodes: [{ group: "plant", edgeNode: "line_1", devices: ["ovenA"] }],
      tags: [],
    };
    expect(MQTT.Task.SCAN_SCHEMAS.statusData.parse(data)).toEqual(data);
  });

  it("should parse the tags of a Sparkplug B browse reply", () => {
    const tag = {
      device: "",
      name: "Node Control/Rebirth",
      dataType: "boolean",
      value: "false",
      supported: true,
    };
    const data = { nodes: [], tags: [tag] };
    expect(MQTT.Task.SCAN_SCHEMAS.statusData.parse(data)).toEqual(data);
  });

  it("should reject a browse reply with no truncated flag", () => {
    expect(z.validate(MQTT.Task.SCAN_SCHEMAS.statusData, { topics: [] })).toBe(false);
  });
});

describe("draft configs", () => {
  // Drafts persist on the Core before configuration, so the shape schema must accept
  // every default config; retrieve parses with it.
  it("should accept the default read config", () => {
    const config = MQTT.Task.READ_SCHEMAS.config.parse({});
    expect(z.validate(MQTT.Task.READ_SCHEMAS.config, config)).toBe(true);
  });

  it("should accept the default write config", () => {
    const config = MQTT.Task.WRITE_SCHEMAS.config.parse({});
    expect(z.validate(MQTT.Task.WRITE_SCHEMAS.config, config)).toBe(true);
  });

  it("should accept the default edge config", () => {
    const config = MQTT.Task.EDGE_SCHEMAS.config.parse({});
    expect(z.validate(MQTT.Task.EDGE_SCHEMAS.config, config)).toBe(true);
  });
});
