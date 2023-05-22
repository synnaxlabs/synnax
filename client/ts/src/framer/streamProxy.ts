import { Stream, EOF } from "@synnaxlabs/freighter";

import { UnexpectedError } from "..";

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
        `${this.name} received unexpected null response from the stream iterator.
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

  send(req: RQ): void {
    const err = this.stream.send(req);
    if (err != null) throw err;
  }
}
