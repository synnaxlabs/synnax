// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Stream, EOF } from "@synnaxlabs/freighter";

import { UnexpectedError } from "@/errors";

export class StreamProxy<RQ, RS> {
  readonly name: string;
  private _stream: Stream<RQ, RS> | null;

  constructor(name: string) {
    this._stream = null;
    this.name = name;
  }

  set stream(stream: Stream<RQ, RS>) {
    this._stream = stream;
  }

  get stream(): Stream<RQ, RS> {
    if (this._stream == null)
      throw new UnexpectedError(
        `Attempted to use uninitialized stream on ${this.name}. 
        This probably means you initialized ${this.name} instead of using the frame client.
        `
      );
    return this._stream;
  }

  async receive(): Promise<RS> {
    const [res, err] = await this.stream.receive();
    if (err != null) throw err;
    if (res == null)
      throw new UnexpectedError(
        `${this.name} received unexpected null response from the stream.
        Please report this error to Synnax team.
      `
      );
    return res;
  }

  received(): boolean {
    return this.stream.received();
  }

  async closeAndAck(): Promise<void> {
    this.stream.closeSend();
    const [, err] = await this.stream.receive();
    if (err == null)
      throw new UnexpectedError(
        `${this.name} received unexpected null error on closure. 
        Please report this error to Synnax team.
      `
      );
    if (!(err instanceof EOF)) throw err;
  }

  closeSend(): void {
    this.stream.closeSend();
  }

  send(req: RQ): void {
    const err = this.stream.send(req);
    if (err != null) throw err;
  }
}
