// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, uuid, zod } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it, onTestFinished } from "vitest";
import { z } from "zod";

import { project } from "@/project";
import { createTestClient } from "@/testutil";

const logEnvelope = (name: string) => ({
  version: 2,
  type: "log",
  name,
  channels: [
    {
      channel: 1,
      color: { r: 127, g: 29, b: 29, a: 1 },
      notation: "scientific",
      precision: 2,
      alias: "temp",
      timestamp: { format: "ISO", tz: "UTC" },
    },
  ],
  timestamp_precision: 1,
  channel_names_hidden: false,
  receipt_timestamp_hidden: true,
});

const toBlob = (value: unknown): Blob => new Blob([JSON.stringify(value)]);

describe("Imex", () => {
  const client = createTestClient();
  let projectKey: project.Key;

  beforeAll(async () => {
    const proj = await client.projects.create({
      name: `imex-proj-${id.create()}`,
    });
    projectKey = proj.key;
  });

  describe("import", () => {
    it("should import from a Blob", async () => {
      const name = `imex-${id.create()}`;
      const resourceID = await client.imex.import(toBlob(logEnvelope(name)), {
        encoding: "JSON",
        fileName: `${name}.json`,
        parent: project.resourceID(projectKey),
      });
      expect(resourceID.type).toEqual("log");
      expect(resourceID.key).not.toHaveLength(0);
    });
    it("should throw an error if the envelope cannot be decoded", async () => {
      const envelope = logEnvelope("invalid");
      envelope.version = -1;
      await expect(
        client.imex.import(toBlob(envelope), {
          encoding: "JSON",
          fileName: "invalid.json",
          parent: project.resourceID(projectKey),
        }),
      ).rejects.toThrow("failed to decode");
    });

    it("should name the resource after the file when the envelope has no name", async () => {
      const { name: _, ...nameless } = logEnvelope("unused");
      const fileName = `imex-file-${id.create()}`;
      const oid = await client.imex.import(toBlob(nameless), {
        encoding: "JSON",
        fileName: `${fileName}.json`,
        parent: project.resourceID(projectKey),
      });
      const stream = await client.imex.export(oid, { encoding: "JSON" });
      const parsed = await new Response(stream).json();
      expect(parsed.name).toEqual(fileName);
    });

    it("should prefer the envelope's name over the file name", async () => {
      const name = `imex-${id.create()}`;
      const oid = await client.imex.import(toBlob(logEnvelope(name)), {
        encoding: "JSON",
        fileName: "Some Other Name.json",
        parent: project.resourceID(projectKey),
      });
      const stream = await client.imex.export(oid, { encoding: "JSON" });
      const parsed = await new Response(stream).json();
      expect(parsed.name).toEqual(name);
    });

    it("should route a typeless legacy state, naming it after the file", async () => {
      // A legacy Console log state: version-stamped, no type, no name. The Core
      // recognizes it by its frozen channels-array marker.
      const state = { version: "0.0.0", channels: [1, 2, 3], remoteCreated: false };
      const fileName = `imex-legacy-${id.create()}`;
      const oid = await client.imex.import(toBlob(state), {
        encoding: "JSON",
        fileName: `${fileName}.json`,
        parent: project.resourceID(projectKey),
      });
      expect(oid.type).toEqual("log");
      const stream = await client.imex.export(oid, { encoding: "JSON" });
      const parsed = await new Response(stream).json();
      expect(parsed.name).toEqual(fileName);
    });

    it("should migrate a legacy task config", async () => {
      // A legacy Console task export: the camelCase config with only a type marker.
      // Its plain routing key moves into a new secret.
      const legacy = {
        type: "pagerduty_alert",
        routingKey: "R016395AF23B4E62B7A2BF7B24C1EF31",
        autoStart: true,
        alerts: [],
      };
      const fileName = `imex-pd-${id.create()}`;
      const oid = await client.imex.import(toBlob(legacy), {
        encoding: "JSON",
        fileName: `${fileName}.json`,
        parent: project.resourceID(projectKey),
      });
      const created = await client.tasks.retrieve({
        key: oid.key,
        schemas: {
          type: z.literal("pagerduty_alert"),
          config: z.looseObject({ routingKey: z.uuid(), autoStart: z.boolean() }),
          statusData: z.unknown(),
        },
      });
      const routingKey = await client.secrets.retrieve(created.config.routingKey);
      onTestFinished(async () => {
        await client.tasks.delete(created.key);
        await client.secrets.delete(routingKey.key);
      });
      expect(created.name).toEqual(fileName);
      expect(created.rack).toBe(0);
      expect(routingKey.name).toBe(`${fileName} routing key`);
      expect(routingKey.source).toBe("stored");
      expect(created.config.autoStart).toBe(true);
    });

    it("should reject an empty file name before the request is sent", async () => {
      const { name: _, ...nameless } = logEnvelope("unused");
      await expect(
        client.imex.import(toBlob(nameless), {
          encoding: "JSON",
          fileName: "",
          parent: project.resourceID(projectKey),
        }),
      ).rejects.toSatisfy(zod.ParseError.matches);
    });

    it("should reject a parent that is not a project before the core stores anything", async () => {
      const name = `imex-${id.create()}`;
      await expect(
        client.imex.import(toBlob(logEnvelope(name)), {
          encoding: "JSON",
          fileName: `${name}.json`,
          parent: { type: "group", key: uuid.create() },
        }),
      ).rejects.toThrow("parent: must be a project");
    });

    it("should put the imported resource in the given project", async () => {
      const proj = await client.projects.create({
        name: `imex-proj-${id.create()}`,
      });
      const name = `imex-${id.create()}`;
      const oid = await client.imex.import(toBlob(logEnvelope(name)), {
        encoding: "JSON",
        fileName: `${name}.json`,
        parent: project.resourceID(proj.key),
      });
      const logs = await client.logs.retrieve({ projects: [proj.key] });
      expect(logs.map(({ key }) => key)).toContain(oid.key);
    });

    it("should put the imported resource in the given group", async () => {
      const proj = await client.projects.create({ name: `imex-proj-${id.create()}` });
      const g = await client.groups.create({
        parent: project.resourceID(proj.key),
        name: `imex-group-${id.create()}`,
      });
      const name = `imex-${id.create()}`;
      const oid = await client.imex.import(toBlob(logEnvelope(name)), {
        encoding: "JSON",
        fileName: `${name}.json`,
        parent: project.resourceID(proj.key),
        group: g.key,
      });
      const l = await client.logs.retrieve({ key: oid.key });
      expect(l.group).toEqual(g.key);
    });

    it("should throw an error when the parent does not exist", async () => {
      const name = `imex-${id.create()}`;
      await expect(
        client.imex.import(toBlob(logEnvelope(name)), {
          encoding: "JSON",
          fileName: `${name}.json`,
          parent: project.resourceID(uuid.create()),
        }),
      ).rejects.toThrow("does not exist");
    });
  });

  describe("export", () => {
    it("should export to a byte stream", async () => {
      const name = `imex-${id.create()}`;
      const oid = await client.imex.import(toBlob(logEnvelope(name)), {
        encoding: "JSON",
        fileName: `${name}.json`,
        parent: project.resourceID(projectKey),
      });
      const stream = await client.imex.export(oid, { encoding: "JSON" });
      const parsed = await new Response(stream).json();
      expect(parsed.type).toEqual("log");
      expect(parsed.name).toEqual(name);
    });
    it("should throw an error if the log is not found", async () => {
      await expect(
        client.imex.export({ type: "log", key: uuid.create() }, { encoding: "JSON" }),
      ).rejects.toThrow("not found");
    });
  });
});
