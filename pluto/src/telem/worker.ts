import { QueryError, UnexpectedError } from "@synnaxlabs/client";
import { TypedWorker } from "@synnaxlabs/x";

import { TelemProvider } from "@/core/vis/telem/TelemService";
import { TelemSourceMeta } from "@/core/vis/telem/TelemSource";
import { CompoundTelemFactory, TelemFactory } from "@/telem/factory";
import { ModifiableTelemSourceMeta } from "@/telem/meta";

interface RemoveMessage {
  variant: "remove";
  key: string;
}

export interface SetMessage {
  variant: "set";
  key: string;
  type: string;
  props: any;
}

export type WorkerMessage = RemoveMessage | SetMessage;

export class TelemService implements TelemProvider {
  factory: TelemFactory;
  telem: Map<string, ModifiableTelemSourceMeta> = new Map();

  constructor(wrap: TypedWorker<WorkerMessage>, factories: TelemFactory[]) {
    this.factory = new CompoundTelemFactory(factories);
    wrap.handle((message) => this.handle(message));
  }

  get<T extends TelemSourceMeta>(key: string): T {
    const v = this.telem.get(key);
    if (v == null)
      throw new QueryError(`Telemetry service could not find source with key ${key}`);
    return v as unknown as T;
  }

  handle(message: WorkerMessage): void {
    const source = this.telem.get(message.key);
    if (message.variant === "remove") {
      if (source == null) {
        console.warn(
          `Telemetry service could not find source with key ${message.key} to remove`
        );
        return;
      }
      source.cleanup();
      this.telem.delete(message.key);
      return;
    }
    if (source == null) this.newSource(message.key, message.type, message.props);
    else source.setProps(message.props);
  }

  newSource(key: string, type: string, props: any): void {
    const source = this.factory.create(key, type, props);
    if (source == null) {
      throw new UnexpectedError(
        `Telemetry service could not find a source for type ${type}`
      );
    }
    this.telem.set(key, source);
  }
}
