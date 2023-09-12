// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type channel,
  Channel,
  DataType,
  QueryError,
  Series,
  TimeRange,
} from "@synnaxlabs/client";
import { bounds, type Destructor, TimeSpan } from "@synnaxlabs/x";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { MockGLBufferController } from "@/mock/MockGLBufferController";
import { client } from "@/telem/client";
import { type telem } from "@/telem/core";
import {
  DynamicXYSource,
  type DynamicXYSourceProps,
  XYSource,
  type XYSourceProps,
} from "@/telem/remote/aether/xy";

import { NumericSource, type NumericSourceProps } from "./numeric";

const X_CHANNEL = new Channel({
  name: "time",
  dataType: DataType.FLOAT32,
  key: 1,
  isIndex: true,
});

const Y_CHANNEL = new Channel({
  name: "data",
  dataType: DataType.FLOAT32,
  key: 2,
  isIndex: false,
  index: 1,
});

const Y_CHANNEL_ALT = new Channel({
  name: "data_alt",
  dataType: DataType.FLOAT32,
  key: 3,
  isIndex: false,
  index: 1,
});

const CHANNELS: Record<channel.Key, Channel> = {
  [X_CHANNEL.key]: X_CHANNEL,
  [Y_CHANNEL.key]: Y_CHANNEL,
  [Y_CHANNEL_ALT.key]: Y_CHANNEL_ALT,
};

describe("XY", () => {
  describe("Static", () => {
    class MockClient implements client.StaticClient, client.ChannelClient {
      data: Record<channel.Key, client.ReadResponse>;
      retrieveChannelMock = vi.fn();
      readMock = vi.fn();

      constructor() {
        const X_CHANNEL_DATA = new Series(
          new Float32Array([1, 2, 3]),
          X_CHANNEL.dataType,
          new TimeRange(0, 10),
        );
        const Y_CHANNEL_DATA = new Series(
          new Float32Array([3, 4, 5]),
          Y_CHANNEL.dataType,
          new TimeRange(0, 10),
        );
        const Y_CHANNEL_ALT_DATA = new Series(
          new Float32Array([6, 7, 8]),
          Y_CHANNEL_ALT.dataType,
          new TimeRange(0, 10),
        );

        this.data = {
          [X_CHANNEL.key]: new client.ReadResponse(X_CHANNEL, [X_CHANNEL_DATA]),
          [Y_CHANNEL.key]: new client.ReadResponse(Y_CHANNEL, [Y_CHANNEL_DATA]),
          [Y_CHANNEL_ALT.key]: new client.ReadResponse(Y_CHANNEL_ALT, [
            Y_CHANNEL_ALT_DATA,
          ]),
        };
      }

      async retrieveChannel(key: number): Promise<Channel> {
        this.retrieveChannelMock(key);
        const ch = CHANNELS[key];
        if (ch == null) throw new QueryError(`Channel ${key} does not exist`);
        return ch;
      }

      async read(
        tr: TimeRange,
        keys: channel.Keys,
      ): Promise<Record<number, client.ReadResponse>> {
        const res: Record<channel.Key, client.ReadResponse> = {};
        keys.forEach((key) => {
          res[key] = this.data[key];
        });
        this.readMock(tr, keys);
        return res;
      }
    }

    const PROPS: XYSourceProps = {
      timeRange: TimeRange.MAX,
      x: X_CHANNEL.key,
      y: Y_CHANNEL.key,
    };

    let telem: telem.XYSource;
    let mockClient: MockClient;
    beforeEach(() => {
      mockClient = new MockClient();
      telem = new XYSource("1", mockClient);
      telem.setProps(PROPS);
    });
    describe("data", () => {
      describe("first read", () => {
        it("should buffer and return the x channel data", async () => {
          const control = new MockGLBufferController();
          const d = await telem.x(control);
          expect(d.length).toBe(1);
          expect(d[0].length).toBe(3);
          // We expect this to be called twice because we predictively buffer the
          // other channel's data as well.
          expect(control.createBufferMock).toHaveBeenCalledTimes(2);
          expect(control.bufferDataMock).toHaveBeenCalledTimes(2);
        });
        it("should buffer and return the y channel data", async () => {
          const control = new MockGLBufferController();
          const d = await telem.y(control);
          expect(d.length).toBe(1);
          expect(d[0].length).toBe(3);
          expect(control.createBufferMock).toHaveBeenCalledTimes(2);
          expect(control.bufferDataMock).toHaveBeenCalledTimes(2);
        });
      });
      describe("second read", () => {
        it("should not re-execute the read on the client", async () => {
          const control = new MockGLBufferController();
          await telem.y(control);
          mockClient.readMock.mockReset();
          mockClient.retrieveChannelMock.mockReset();
          const d2 = await telem.y(control);
          expect(d2.length).toBe(1);
          expect(mockClient.readMock).not.toHaveBeenCalled();
          expect(mockClient.retrieveChannelMock).not.toHaveBeenCalled();
        });
        it("shouild not increment the series reference count", async () => {
          const control = new MockGLBufferController();
          await telem.y(control);
          const d2 = await telem.y(control);
          expect(d2[0].refCount).toEqual(1);
        });
      });
    });
    describe("bounds", () => {
      it("should return the bounds of the x channel's data", async () => {
        const bounds = await telem.xBounds();
        expect(bounds).toEqual({ lower: 1, upper: 3 });
      });
      it("should return the bounds of the y channel's data", async () => {
        const bounds = await telem.yBounds();
        expect(bounds).toEqual({ lower: 3, upper: 5 });
      });
    });
    describe("invalidate", () => {
      it("should delete the WebGL buffers", async () => {
        const control = new MockGLBufferController();
        await telem.x(control);
        telem.invalidate();
        expect(control.deleteBufferMock).toHaveBeenCalled();
      });
      it("should notify an onchange handler", async () => {
        const notify = vi.fn();
        telem.onChange(notify);
        telem.invalidate();
        expect(notify).toHaveBeenCalledTimes(1);
      });
    });
    describe("setProps", () => {
      it("should invalidate the data if the props change", async () => {
        const control = new MockGLBufferController();
        await telem.x(control);
        const props: XYSourceProps = {
          x: 1,
          y: 3,
          timeRange: TimeRange.MAX,
        };
        telem.setProps(props);
        mockClient.readMock.mockReset();
        mockClient.retrieveChannelMock.mockReset();
        const d2 = await telem.y(control);
        expect(d2).toHaveLength(1);
        expect(d2[0].at(0)).toEqual(6);
        expect(mockClient.readMock).toHaveBeenCalledTimes(1);
        expect(mockClient.retrieveChannelMock).toHaveBeenCalledTimes(2);
      });
      it("should return cached data if the props do not change", async () => {
        const control = new MockGLBufferController();
        await telem.x(control);
        telem.setProps(PROPS);
        mockClient.readMock.mockReset();
        mockClient.retrieveChannelMock.mockReset();
        const d2 = await telem.y(control);
        expect(d2).toHaveLength(1);
        expect(mockClient.readMock).not.toHaveBeenCalled();
        expect(mockClient.retrieveChannelMock).not.toHaveBeenCalled();
      });
    });
  });

  class MockStreamClient implements client.Client {
    handler: client.StreamHandler | undefined = undefined;

    async retrieveChannel(key: number): Promise<Channel> {
      return CHANNELS[key];
    }

    async read(
      tr: TimeRange,
      keys: channel.Keys,
    ): Promise<Record<number, client.ReadResponse>> {
      return {
        [X_CHANNEL.key]: new client.ReadResponse(X_CHANNEL, []),
        [Y_CHANNEL.key]: new client.ReadResponse(Y_CHANNEL, []),
      };
    }

    async stream(
      handler: client.StreamHandler,
      keys: channel.Keys,
    ): Promise<Destructor> {
      this.handler = handler;
      return () => {
        this.handler = undefined;
      };
    }

    close(): void {}
  }

  describe("Dynamic", () => {
    const PROPS: DynamicXYSourceProps = {
      x: 1,
      y: 2,
      span: TimeSpan.MAX,
    };

    let telem: telem.XYSource;
    let client_: MockStreamClient;
    beforeEach(() => {
      client_ = new MockStreamClient();
      telem = new DynamicXYSource("1", client_);
      telem.setProps(PROPS);
    });

    describe("read", () => {
      it("should bind a stream handler on data request", async () => {
        const control = new MockGLBufferController();
        await telem.x(control);
        expect(client_.handler).toBeDefined();
      });
      it("should update its internal buffer when the stream handler changes", async () => {
        const control = new MockGLBufferController();
        await telem.x(control);
        client_.handler?.({
          1: new client.ReadResponse(X_CHANNEL, [
            new Series(new Float32Array([1, 2, 3])),
          ]),
          2: new client.ReadResponse(Y_CHANNEL, [
            new Series(new Float32Array([4, 5, 6])),
          ]),
        });
        control.createBufferMock.mockReset();
        await telem.x(control);
        expect(control.createBufferMock).toHaveBeenCalledTimes(2);
      });
    });
  });
  describe("Numeric", () => {
    const PROPS: NumericSourceProps = {
      channel: 1,
    };
    let telem_: telem.NumericSource;
    let client_: MockStreamClient;
    beforeEach(() => {
      client_ = new MockStreamClient();
      telem_ = new NumericSource("1", client_);
      telem_.setProps(PROPS);
    });

    describe("read", () => {
      it("should return zero if no current value exists", async () => {
        expect(await telem_.value()).toBe(0);
      });
      it("should update the value when the stream handler changes", async () => {
        expect(await telem_.value()).toBe(0);
        client_.handler?.({
          1: new client.ReadResponse(CHANNELS[1], [new Series(new Float32Array([1]))]),
        });
        expect(await telem_.value()).toBe(1);
      });
    });
  });
});
