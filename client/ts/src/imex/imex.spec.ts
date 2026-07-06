// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, uuid } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it } from "vitest";

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
  hide_channel_names: false,
  hide_receipt_timestamp: true,
});

const toBlob = (value: unknown): Blob => new Blob([JSON.stringify(value)]);

describe("Imex", () => {
  const client = createTestClient();
  let projectKey: project.Key;

  beforeAll(async () => {
    const proj = await client.projects.create({
      name: `imex-proj-${id.create()}`,
      layout: {},
    });
    projectKey = proj.key;
  });

  describe("import", () => {
    it("should import from a Blob", async () => {
      const name = `imex-${id.create()}`;
      const ontologyID = await client.imex.import(toBlob(logEnvelope(name)), {
        encoding: "JSON",
        fileName: `${name}.json`,
        project: projectKey,
      });
      expect(ontologyID.type).toEqual("log");
      expect(ontologyID.key).not.toHaveLength(0);
    });
    it("should throw an error if the envelope cannot be decoded", async () => {
      const envelope = logEnvelope("invalid");
      envelope.version = -1;
      await expect(
        client.imex.import(toBlob(envelope), {
          encoding: "JSON",
          fileName: "invalid.json",
          project: projectKey,
        }),
      ).rejects.toThrow("failed to decode");
    });

    it("should name the resource after the file when the envelope has no name", async () => {
      const { name: _, ...nameless } = logEnvelope("unused");
      const fileName = `imex-file-${id.create()}`;
      const oid = await client.imex.import(toBlob(nameless), {
        encoding: "JSON",
        fileName: `${fileName}.json`,
        project: projectKey,
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
        project: projectKey,
      });
      const stream = await client.imex.export(oid, { encoding: "JSON" });
      const parsed = await new Response(stream).json();
      expect(parsed.name).toEqual(name);
    });

    it("should throw an error when the envelope has no name and the file name is empty", async () => {
      const { name: _, ...nameless } = logEnvelope("unused");
      await expect(
        client.imex.import(toBlob(nameless), {
          encoding: "JSON",
          fileName: "",
          project: projectKey,
        }),
      ).rejects.toThrow("name");
    });

    it("should parent the imported resource under the given parent", async () => {
      const proj = await client.projects.create({
        name: `imex-proj-${id.create()}`,
        layout: {},
      });
      const name = `imex-${id.create()}`;
      const oid = await client.imex.import(toBlob(logEnvelope(name)), {
        encoding: "JSON",
        fileName: `${name}.json`,
        project: proj.key,
      });
      const children = await client.ontology.retrieveChildren(
        project.ontologyID(proj.key),
      );
      expect(children.map((child) => child.id.key)).toContain(oid.key);
    });

    it("should throw an error when the parent does not exist", async () => {
      const name = `imex-${id.create()}`;
      await expect(
        client.imex.import(toBlob(logEnvelope(name)), {
          encoding: "JSON",
          fileName: `${name}.json`,
          project: uuid.create(),
        }),
      ).rejects.toThrow("not found");
    });
  });

  describe("export", () => {
    it("should export to a byte stream", async () => {
      const name = `imex-${id.create()}`;
      const oid = await client.imex.import(toBlob(logEnvelope(name)), {
        encoding: "JSON",
        fileName: `${name}.json`,
        project: projectKey,
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
