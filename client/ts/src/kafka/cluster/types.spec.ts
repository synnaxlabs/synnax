// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { kafka } from "@/kafka";

const CLUSTER = {
  key: "22222222-2222-4222-8222-222222222222",
  rack: 1,
  name: "Plant Cluster",
  brokers: [{ host: "kafka.example.com" }],
} as const;

const SECRET = "33333333-3333-4333-8333-333333333333";

describe("clusterZ", () => {
  it("should parse a cluster and apply the defaults", () => {
    const cluster = kafka.cluster.clusterZ.parse(CLUSTER);
    expect(cluster.brokers).toEqual([{ host: "kafka.example.com" }]);
    expect(cluster.transport).toEqual({ type: "plaintext" });
    expect(cluster.auth).toEqual({ mechanism: "none" });
    expect(cluster.reads).toEqual([]);
  });

  it("should apply the defaults of a read", () => {
    const cluster = kafka.cluster.clusterZ.parse({
      ...CLUSTER,
      reads: [{ topic: "plant", message: SECRET, index: 1, channel: 2 }],
    });
    expect(cluster.reads[0].recordKey).toBeUndefined();
    expect(cluster.reads[0].pointer).toEqual("");
  });

  it("should parse SCRAM credentials that name their password by secret", () => {
    const cluster = kafka.cluster.clusterZ.parse({
      ...CLUSTER,
      transport: { type: "tls", clientCertificate: SECRET },
      auth: { mechanism: "scram_sha_256", username: "admin", password: SECRET },
    });
    expect(cluster.transport).toEqual({ type: "tls", clientCertificate: SECRET });
    expect(cluster.auth).toEqual({
      mechanism: "scram_sha_256",
      username: "admin",
      password: SECRET,
    });
  });

  it("should parse a read of one record key", () => {
    const cluster = kafka.cluster.clusterZ.parse({
      ...CLUSTER,
      reads: [
        { topic: "plant", recordKey: "pump", message: SECRET, index: 1, channel: 2 },
      ],
    });
    expect(cluster.reads[0].recordKey).toEqual("pump");
  });

  const INVALID: [string, Record<string, unknown>][] = [
    ["an empty name", { name: "" }],
    ["an empty host", { brokers: [{ host: "" }] }],
    ["port 0", { brokers: [{ host: "kafka.example.com", port: 0 }] }],
    ["an unknown transport", { transport: { type: "ssl" } }],
    [
      "a plaintext password",
      { auth: { mechanism: "plain", username: "a", password: "x" } },
    ],
    [
      "credentials with no user name",
      { auth: { mechanism: "plain", password: SECRET } },
    ],
    [
      "a read with an empty record key",
      {
        reads: [{ topic: "a", recordKey: "", message: SECRET, index: 1, channel: 2 }],
      },
    ],
    [
      "a read with no index",
      { reads: [{ topic: "a", message: SECRET, index: 0, channel: 2 }] },
    ],
  ];
  it.each(INVALID)("should reject %s", (_, change) => {
    expect(kafka.cluster.clusterZ.safeParse({ ...CLUSTER, ...change }).success).toBe(
      false,
    );
  });
});
