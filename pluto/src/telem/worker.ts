import { TypedWorker } from "@synnaxlabs/x";

import { StaticTelemFactory } from "./staticTelem";

import { TelemProvider } from "@/core/vis/telem/TelemService";
import { TelemSourceMeta } from "@/core/vis/telem/TelemSource";

export interface WorkerMessage {
  key: string;
  type: string;
  props: any;
}

export interface BaseTelemSourceMeta extends TelemSourceMeta {
  setProps: (props: any) => void;
}

export class TelemWorker implements TelemProvider {
  telem: Map<string, BaseTelemSourceMeta> = new Map();
  staticFactory: StaticTelemFactory;

  constructor(wrap: TypedWorker<WorkerMessage>) {
    wrap.handle((message) => this.handle(message));
    this.staticFactory = new StaticTelemFactory();
  }

  get<T extends TelemSourceMeta>(key: string): T {
    // @ts-expect-error
    return this.telem.get(key) as T;
  }

  handle(message: WorkerMessage): void {
    const source = this.telem.get(message.key);
    if (source == null) this.newSource(message.key, message.type, message.props);
    else source.setProps(message.props);
  }

  newSource(key: string, type: string, props: any): void {
    if (type !== "static") throw new Error(`Unknown telem source type: ${type}`);
    const n = this.staticFactory.new(key, props);
    this.telem.set(key, n);
  }
}
