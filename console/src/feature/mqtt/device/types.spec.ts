// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { caseconv } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { MQTT } from "@/feature/mqtt";

describe("MQTT Device Properties", () => {
  describe("propertiesZ", () => {
    it("should validate a TLS broker with client certificates", () => {
      const result = MQTT.Device.propertiesZ.parse({
        ...MQTT.Device.ZERO_PROPERTIES,
        secure: true,
        port: 8883,
        caFile: "/etc/ssl/ca.pem",
        certFile: "/etc/ssl/client.pem",
        keyFile: "/etc/ssl/client.key",
      });
      expect(result.secure).toBe(true);
      expect(result.keyFile).toBe("/etc/ssl/client.key");
    });

    it("should leave the port and keep alive absent, so the Core picks them", () => {
      expect(MQTT.Device.ZERO_PROPERTIES).not.toHaveProperty("port");
      expect(MQTT.Device.ZERO_PROPERTIES).not.toHaveProperty("keepAlive");
    });

    it.each([0, -1, 65536])("should reject the port %d", (port) => {
      const result = MQTT.Device.propertiesZ.safeParse({ port });
      expect(result.error?.issues).toMatchObject([
        { message: "Port must be 1 to 65535", path: ["port"] },
      ]);
    });

    it.each([0, -1, 65536])("should reject the keep alive %d", (keepAlive) => {
      const result = MQTT.Device.propertiesZ.safeParse({ keepAlive });
      expect(result.error?.issues).toMatchObject([
        { message: "Keep alive must be 1 to 65535", path: ["keepAlive"] },
      ]);
    });

    it.each([1, 65535])("should accept a port and a keep alive of %d", (value) => {
      const result = MQTT.Device.propertiesZ.parse({ port: value, keepAlive: value });
      expect(result).toMatchObject({ port: value, keepAlive: value });
    });

    it("should reject a version other than 1", () => {
      expect(MQTT.Device.propertiesZ.safeParse({ version: 2 }).success).toBe(false);
    });

    it("should accept a Sparkplug B host ID and groups", () => {
      const sparkplug = { hostId: "synnax_core", groups: ["plant", "utilities"] };
      expect(MQTT.Device.propertiesZ.parse({ sparkplug }).sparkplug).toEqual(sparkplug);
    });

    it.each(["a/b", "host+", "#"])("should reject the host ID %s", (hostId) => {
      const result = MQTT.Device.propertiesZ.safeParse({ sparkplug: { hostId } });
      expect(result.error?.issues).toMatchObject([
        { message: "Host ID must not hold /, +, or #", path: ["sparkplug", "hostId"] },
      ]);
    });

    it("should put the issue of a bad group on the list of groups", () => {
      const result = MQTT.Device.propertiesZ.safeParse({
        sparkplug: { groups: ["plant", "line/1", ""] },
      });
      expect(result.error?.issues).toMatchObject([
        {
          message: 'Group "line/1" must not hold /, +, or #',
          path: ["sparkplug", "groups"],
        },
        { message: "Group must not be empty", path: ["sparkplug", "groups"] },
      ]);
    });

    it("should read a stored index that is not a channel key as no index", () => {
      const result = MQTT.Device.propertiesZ.parse({
        read: { "plant/temp": { index: "gone", channels: {} } },
      });
      expect(result.read["plant/temp"].index).toBe(0);
    });
  });

  describe("wire format", () => {
    const properties: MQTT.Device.Properties = {
      ...MQTT.Device.ZERO_PROPERTIES,
      clientId: "console",
      sparkplug: { hostId: "synnaxCore_1", groups: ["plant_A"] },
      read: {
        "plant/line_a/tempC": { index: 1, channels: { "/outletTemp_c": 2 } },
        "spBv1.0/plant_A/lineOne/oven_temp": { index: 4, channels: { "": 5 } },
      },
      write: { "plant/line_a/setPoint": 3, "spBv1.0/plant_A/lineOne/setPoint": 6 },
    };
    const schema = MQTT.Device.propertiesZ;

    it("should send the connection properties in the snake case the Core reads", () => {
      const wire = caseconv.camelToSnake({ ...properties, keepAlive: 45 }, { schema });
      expect(wire).toMatchObject({ client_id: "console", keep_alive: 45, ca_file: "" });
    });

    it("should leave an unset port and keep alive off the wire", () => {
      const wire = caseconv.camelToSnake(properties, { schema });
      expect(wire).not.toHaveProperty("port");
      expect(wire).not.toHaveProperty("keep_alive");
    });

    it("should send the Sparkplug B host ID under a snake case key", () => {
      const wire = caseconv.camelToSnake(properties, { schema });
      expect(wire).toMatchObject({
        sparkplug: { host_id: "synnaxCore_1", groups: ["plant_A"] },
      });
    });

    it("should keep the case of topics and pointers through a round trip", () => {
      const wire = caseconv.camelToSnake(properties, { schema });
      expect(caseconv.snakeToCamel(wire, { schema })).toEqual(properties);
      expect(wire).toMatchObject({ read: properties.read, write: properties.write });
    });
  });

  describe("SCHEMAS", () => {
    it("should accept the mqtt make and the MQTT broker model", () => {
      expect(MQTT.Device.SCHEMAS.make.parse(MQTT.Device.MAKE)).toBe("mqtt");
      expect(MQTT.Device.SCHEMAS.model.parse("MQTT broker")).toBe("MQTT broker");
    });

    it("should reject another make", () => {
      expect(MQTT.Device.SCHEMAS.make.safeParse("http").success).toBe(false);
    });
  });
});
