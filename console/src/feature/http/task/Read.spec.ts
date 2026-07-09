// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { HTTP } from "@/feature/http";
import { createHTTPDevice } from "@/feature/http/testutil";
import { type Task } from "@/platform/task";
import {
  awaitTaskKey,
  clickConfigure,
  renderTaskFormView,
} from "@/platform/task/testutil";
import { getHeaderIconButton, uniqueName } from "@/testutil";

const renderRead = async (
  options: { client?: Synnax | null; args?: Task.FormViewArgs } = {},
) => await renderTaskFormView(HTTP.Task.Read, HTTP.Task.READ_TYPE, options);

const addEndpoint = async (): Promise<void> => {
  fireEvent.click(await screen.findByText("Add an endpoint"));
  await screen.findByText("Timing mode");
};

const createReadField = (
  key: string,
  pointer: string,
  overrides: Partial<HTTP.Task.ReadField> = {},
): HTTP.Task.ReadField => ({
  ...HTTP.Task.ZERO_READ_FIELD,
  key,
  pointer,
  ...overrides,
});

const createReadConfig = (
  device: string,
  endpoints: HTTP.Task.ReadEndpoint[],
): HTTP.Task.ReadPayload["config"] => ({
  ...HTTP.Task.ZERO_READ_PAYLOAD.config,
  device,
  endpoints,
});

const configureAndAwaitTask = async (
  client: Synnax,
  rendered: Awaited<ReturnType<typeof renderRead>>,
) => {
  await clickConfigure();
  const taskKey = await awaitTaskKey(rendered);
  return await client.tasks.retrieve({ key: taskKey, schemas: HTTP.Task.READ_SCHEMAS });
};

describe("HTTP Read form", () => {
  it("should show the empty state and add + select an endpoint", async () => {
    await renderRead();
    await screen.findByText("Select an endpoint to configure");
    await screen.findByText("No endpoints.");
    await addEndpoint();
    expect(screen.getByRole("button", { name: "GET" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "POST" })).toBeTruthy();
    expect(screen.getByPlaceholderText("/api/data")).toBeTruthy();
    expect(screen.getByText("Headers")).toBeTruthy();
    expect(screen.getByText("Query parameters")).toBeTruthy();
    expect(screen.getByText("No fields.")).toBeTruthy();
    expect(screen.queryByText("Select an endpoint to configure")).toBeNull();
  });

  it("should reveal the request body field when the method switches to POST", async () => {
    await renderRead();
    await addEndpoint();
    expect(screen.queryByPlaceholderText('{"query": "latest"}')).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "POST" }));
    await screen.findByPlaceholderText('{"query": "latest"}');
    fireEvent.click(screen.getByRole("button", { name: "GET" }));
    await waitFor(() =>
      expect(screen.queryByPlaceholderText('{"query": "latest"}')).toBeNull(),
    );
  });

  it("should add a timestamp field on value timing that stays out of the fields list", async () => {
    await renderRead();
    await addEndpoint();
    fireEvent.click(screen.getByRole("button", { name: "Value" }));
    await screen.findByText("Timestamp pointer");
    expect(screen.getByText("Format")).toBeTruthy();
    expect(screen.getByText("No fields.")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Software" }));
    await waitFor(() => expect(screen.queryByText("Timestamp pointer")).toBeNull());
  });

  it("should add a field, select it, and show the enum mapping editor", async () => {
    await renderRead();
    await addEndpoint();
    fireEvent.click(screen.getByText("Add a field"));
    await screen.findByPlaceholderText("/temperature");
    await screen.findByText("Enum mapping");
  });

  it("should copy the previous field's settings when adding another field", async () => {
    await renderRead();
    await addEndpoint();
    fireEvent.click(screen.getByText("Add a field"));
    const pointer = await screen.findByPlaceholderText("/temperature");
    fireEvent.change(pointer, { target: { value: "/a" } });
    fireEvent.blur(pointer);
    fireEvent.click(getHeaderIconButton("Fields"));
    await waitFor(() => expect(screen.getAllByDisplayValue("/a")).toHaveLength(2));
  });

  it("should duplicate and delete endpoints through the context menu", async () => {
    await renderRead();
    await addEndpoint();
    const path = screen.getByPlaceholderText("/api/data");
    fireEvent.change(path, { target: { value: "/api/v1" } });
    fireEvent.blur(path);
    const item = await screen.findByText(/\/api\/v1/);
    fireEvent.contextMenu(item);
    fireEvent.click(await screen.findByText("Duplicate"));
    await waitFor(() => expect(screen.getAllByText(/\/api\/v1/)).toHaveLength(2));
    fireEvent.contextMenu(screen.getAllByText(/\/api\/v1/)[0]);
    fireEvent.click(await screen.findByText("Delete"));
    await waitFor(() => expect(screen.getAllByText(/\/api\/v1/)).toHaveLength(1));
  });

  it("should seed the form from a config passed through view args", async () => {
    const config = createReadConfig("dev_1", [
      {
        ...HTTP.Task.ZERO_READ_ENDPOINT,
        key: "ep1",
        path: "/seeded",
        fields: [createReadField("f1", "/temp")],
      },
    ]);
    await renderRead({ args: { config } });
    await screen.findByText(/\/seeded/);
  });

  describe("onConfigure against a live cluster", () => {
    const client = createTestClient();

    it("should create index, data, and virtual channels and persist them to the device", async () => {
      const dev = await createHTTPDevice(client);
      const config = createReadConfig(dev.key, [
        {
          ...HTTP.Task.ZERO_READ_ENDPOINT,
          key: "ep1",
          path: "/data",
          index: "tf",
          fields: [
            createReadField("f1", "/temperature"),
            createReadField("f2", "/label", { dataType: "string" }),
            createReadField("tf", "/ts", { timestampFormat: "unix_sec" }),
          ],
        },
      ]);
      const rendered = await renderRead({ client, args: { config } });
      const created = await configureAndAwaitTask(client, rendered);

      const updated = await client.devices.retrieve({
        key: dev.key,
        schemas: HTTP.Device.SCHEMAS,
      });
      const epProps = updated.properties.read["/data"];
      expect(epProps.index).toBeGreaterThan(0);

      const dataKey = epProps.channels["/temperature"];
      const dataCh = await client.channels.retrieve(dataKey);
      expect(dataCh.index).toBe(epProps.index);

      const virtualKey = epProps.channels["/label"];
      const virtualCh = await client.channels.retrieve(virtualKey);
      expect(virtualCh.virtual).toBe(true);

      const fields = created.config.endpoints[0].fields;
      expect(fields.find((f) => f.key === "f1")?.channel).toBe(dataKey);
      expect(fields.find((f) => f.key === "f2")?.channel).toBe(virtualKey);
      expect(fields.find((f) => f.key === "tf")?.channel).toBe(epProps.index);
    });

    it("should reuse channels already stored on the device instead of creating new ones", async () => {
      const dev = await createHTTPDevice(client);
      const idxCh = await client.channels.create({
        name: uniqueName("http_idx"),
        dataType: "timestamp",
        isIndex: true,
      });
      const dataCh = await client.channels.create({
        name: uniqueName("http_data"),
        dataType: "float64",
        index: idxCh.key,
      });
      dev.properties = {
        ...HTTP.Device.ZERO_PROPERTIES,
        read: {
          "/data": { index: idxCh.key, channels: { "/temperature": dataCh.key } },
        },
      };
      await client.devices.create(dev);
      const config = createReadConfig(dev.key, [
        {
          ...HTTP.Task.ZERO_READ_ENDPOINT,
          key: "ep1",
          path: "/data",
          fields: [createReadField("f1", "/temperature")],
        },
      ]);
      const rendered = await renderRead({ client, args: { config } });
      const created = await configureAndAwaitTask(client, rendered);
      expect(created.config.endpoints[0].fields[0].channel).toBe(dataCh.key);
    });

    it("should recover the index from an existing data channel when the stored index is gone", async () => {
      const dev = await createHTTPDevice(client);
      const idxCh = await client.channels.create({
        name: uniqueName("http_idx"),
        dataType: "timestamp",
        isIndex: true,
      });
      const dataCh = await client.channels.create({
        name: uniqueName("http_data"),
        dataType: "float64",
        index: idxCh.key,
      });
      dev.properties = {
        ...HTTP.Device.ZERO_PROPERTIES,
        read: { "/data": { index: 0, channels: { "/existing": dataCh.key } } },
      };
      await client.devices.create(dev);
      const config = createReadConfig(dev.key, [
        {
          ...HTTP.Task.ZERO_READ_ENDPOINT,
          key: "ep1",
          path: "/data",
          fields: [createReadField("f1", "/temperature")],
        },
      ]);
      const rendered = await renderRead({ client, args: { config } });
      await configureAndAwaitTask(client, rendered);
      const updated = await client.devices.retrieve({
        key: dev.key,
        schemas: HTTP.Device.SCHEMAS,
      });
      expect(updated.properties.read["/data"].index).toBe(idxCh.key);
    });
  });
});
