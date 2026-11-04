// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Kafka } from "@/feature/kafka";

describe("Kafka Device Properties", () => {
  it("should accept a cluster with brokers and no authentication", () => {
    const props = Kafka.Device.SCHEMAS.properties.parse({
      brokers: ["localhost:9092"],
      tls: false,
      sasl: { mechanism: "none" },
    });
    expect(props.brokers).toEqual(["localhost:9092"]);
  });

  it("should accept SASL credentials", () => {
    const props = Kafka.Device.SCHEMAS.properties.parse({
      brokers: ["broker:9093"],
      tls: true,
      sasl: { mechanism: "scram_sha_256", username: "u", password: "p" },
    });
    expect(props.sasl.mechanism).toBe("scram_sha_256");
  });

  it("should reject an empty broker address", () => {
    const result = Kafka.Device.SCHEMAS.properties.safeParse(
      Kafka.Device.ZERO_PROPERTIES,
    );
    expect(result.success).toBe(false);
    expect(result.error?.issues[0].message).toBe("Broker address is required");
  });

  it("should reject a cluster without brokers", () => {
    const result = Kafka.Device.SCHEMAS.properties.safeParse({ brokers: [] });
    expect(result.success).toBe(false);
    expect(result.error?.issues[0].message).toBe("At least one broker is required");
  });
});
