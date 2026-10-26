// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Kafka } from "@/feature/kafka";
import { createKafkaDevice } from "@/feature/kafka/testutil";
import { createTestSink } from "@/platform/analytics/testutil";
import { createRunningRack } from "@/platform/device/testutil";
import { pressSaveTrigger, renderModalOpener } from "@/platform/modals/testutil";
import { answerNextCommand, selectFromDropdown } from "@/platform/task/testutil";
import { getSwitchInput, uniqueName } from "@/testutil";

const client = createTestClient();

describe("Kafka.Device.useConnectModal", () => {
  it("should populate the form from an existing device", async () => {
    const dev = await createKafkaDevice(client, {
      properties: {
        brokers: ["broker-a:9092", "broker-b:9092"],
        tls: true,
        sasl: { mechanism: "plain", username: "alice", password: "secret" },
      },
    });
    await renderModalOpener(Kafka.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    await screen.findByDisplayValue(dev.name);
    expect(screen.getByDisplayValue("broker-a:9092")).toBeTruthy();
    expect(screen.getByDisplayValue("broker-b:9092")).toBeTruthy();
    expect(getSwitchInput("TLS").checked).toBe(true);
    expect(await screen.findByDisplayValue("alice")).toBeTruthy();
    expect(screen.getByDisplayValue("secret")).toBeTruthy();
  });

  it("should add and remove broker inputs", async () => {
    await renderModalOpener(Kafka.Device.useConnectModal, [{}], { client });
    await screen.findByRole("dialog");
    expect(screen.getAllByPlaceholderText("localhost:9092")).toHaveLength(1);
    fireEvent.click(screen.getByRole("button", { name: "Add broker" }));
    await waitFor(() =>
      expect(screen.getAllByPlaceholderText("localhost:9092")).toHaveLength(2),
    );
    fireEvent.click(screen.getAllByRole("button", { name: "Remove broker" })[0]);
    await waitFor(() =>
      expect(screen.getAllByPlaceholderText("localhost:9092")).toHaveLength(1),
    );
  });

  it("should reveal the credential fields for a SASL mechanism", async () => {
    await renderModalOpener(Kafka.Device.useConnectModal, [{}], { client });
    await screen.findByRole("dialog");
    expect(screen.queryByPlaceholderText("username")).toBeNull();
    await selectFromDropdown("None", "SCRAM-SHA-256");
    await screen.findByPlaceholderText("username");
    await selectFromDropdown("SCRAM-SHA-256", "None");
    await waitFor(() => expect(screen.queryByPlaceholderText("username")).toBeNull());
  });

  it("should start with an empty, focused name", async () => {
    await renderModalOpener(Kafka.Device.useConnectModal, [{}], { client });
    const name = await screen.findByPlaceholderText("Kafka cluster");
    expect(name).toHaveProperty("value", "");
    expect(document.activeElement).toBe(name);
  });

  it("should show the CA certificate only with TLS on and clear it when TLS turns off", async () => {
    await renderModalOpener(Kafka.Device.useConnectModal, [{}], { client });
    await screen.findByRole("dialog");
    const caPlaceholder = "-----BEGIN CERTIFICATE-----";
    expect(screen.queryByPlaceholderText(caPlaceholder)).toBeNull();
    fireEvent.click(getSwitchInput("TLS"));
    const ca = await screen.findByPlaceholderText(caPlaceholder);
    fireEvent.change(ca, { target: { value: "pem" } });
    fireEvent.click(getSwitchInput("TLS"));
    await waitFor(() =>
      expect(screen.queryByPlaceholderText(caPlaceholder)).toBeNull(),
    );
    fireEvent.click(getSwitchInput("TLS"));
    expect(await screen.findByPlaceholderText(caPlaceholder)).toHaveProperty(
      "value",
      "",
    );
  });

  it("should report a device it connected", async () => {
    const rack = await createRunningRack(client, "kafka");
    const scan = await rack.createTask({
      name: uniqueName("kafka_scan"),
      type: Kafka.Task.SCAN_TYPE,
      config: {},
    });
    const analytics = createTestSink();
    await renderModalOpener(Kafka.Device.useConnectModal, [{ rack: rack.key }], {
      client,
      analytics,
    });
    fireEvent.change(screen.getByPlaceholderText("Kafka cluster"), {
      target: { value: uniqueName("kafka_cluster") },
    });
    fireEvent.change(screen.getByPlaceholderText("localhost:9092"), {
      target: { value: "localhost:9092" },
    });
    const { answered } = await answerNextCommand(client, scan);
    pressSaveTrigger();
    await answered;
    await waitFor(() =>
      expect(analytics.capture).toHaveBeenCalledWith("device_connected", {
        integration: "kafka",
      }),
    );
  });

  // A running rack keeps Connect enabled while none is chosen, so the save fails
  // validation. That error proves the keys reached the Connect button's save path.
  it("should submit on the shortcut its footer advertises", async () => {
    await createRunningRack(client, "kafka");
    await renderModalOpener(Kafka.Device.useConnectModal, [{}], { client });
    await screen.findByRole("dialog");
    pressSaveTrigger();
    expect(await screen.findByText(/rack is required/i)).toBeTruthy();
  });
});
