// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, DataType } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/channel";

const createPayload = (fields: Partial<channel.Payload>): channel.Payload =>
  channel.payloadZ.parse({
    key: 2,
    name: "pressure",
    dataType: DataType.FLOAT64,
    ...fields,
  });

const TIME = createPayload({
  key: 1,
  name: "time",
  dataType: DataType.TIMESTAMP,
  isIndex: true,
});

const factsOf = (container: HTMLElement): string | null | undefined =>
  container.querySelector(".lyra-haul-preview__facts")?.textContent;

describe("Channel.HaulPreview", () => {
  it("should show the name, the data type, and the index", () => {
    const { container } = render(
      <Channel.HaulPreview
        channel={createPayload({ index: 1 })}
        index={TIME}
        calculated={false}
      />,
    );
    expect(screen.getByText("pressure")).toBeDefined();
    expect(factsOf(container)).toBe("float64·time");
  });

  it("should show the data type alone when the index is not given", () => {
    const { container } = render(
      <Channel.HaulPreview channel={createPayload({ index: 1 })} calculated={false} />,
    );
    expect(factsOf(container)).toBe("float64");
  });

  it("should name an index channel as an index", () => {
    const { container } = render(
      <Channel.HaulPreview channel={TIME} calculated={false} />,
    );
    expect(factsOf(container)).toBe("Timestamp·Index");
  });

  it("should name a channel that a calculation writes as calculated", () => {
    const output = createPayload({ virtual: true });
    const { container } = render(<Channel.HaulPreview channel={output} calculated />);
    expect(factsOf(container)).toBe("float64·Calculated");
  });

  it("should name a virtual channel as virtual", () => {
    const virtual = createPayload({ virtual: true, dataType: DataType.FLOAT32 });
    const { container } = render(
      <Channel.HaulPreview channel={virtual} calculated={undefined} />,
    );
    expect(factsOf(container)).toBe("float32·Virtual");
  });

  it("should stack with a count when the drag carries many channels", () => {
    const { container } = render(
      <Channel.HaulPreview channel={TIME} count={4} calculated={false} />,
    );
    expect(container.querySelector(".lyra-haul-preview__count")?.textContent).toBe("4");
  });
});

describe("Channel.getCachedPreviewProps", () => {
  const client = createTestClient();

  it("should return the index of a channel from the cache", async () => {
    const index = await client.channels.create({
      name: id.create(),
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    const data = await client.channels.create({
      name: id.create(),
      dataType: DataType.FLOAT64,
      index: index.key,
    });
    expect(Channel.getCachedPreviewProps(client, data.payload).index?.key).toBe(
      index.key,
    );
  });

  it("should return no index for a channel with no index", () => {
    expect(Channel.getCachedPreviewProps(client, TIME).index).toBeUndefined();
  });

  it("should tell an output from a plain channel once the cache holds the answers", async () => {
    const calc = await client.calculations.create({
      name: id.create(),
      expression: "return 2",
    });
    const plain = await client.channels.create({
      name: id.create(),
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    const [output] = await client.channels.retrieve([calc.output]);
    const calculated = (ch: channel.Channel): boolean | undefined =>
      Channel.getCachedPreviewProps(client, ch.payload).calculated;
    expect(calculated(plain)).toBeUndefined();
    await client.calculations.outputs.retrieve([plain.key, calc.output]);
    expect(calculated(output)).toBe(true);
    expect(calculated(plain)).toBe(false);
  });

  it("should return nothing without a client", () => {
    expect(Channel.getCachedPreviewProps(null, createPayload({ index: 1 }))).toEqual({
      index: undefined,
      calculated: undefined,
    });
  });
});
