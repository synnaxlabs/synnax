import { TelemSourceMeta } from "./TelemSource";

export interface TelemProvider {
  get: <T extends TelemSourceMeta>(key: string) => T;
}

export interface TelemFactory {
  new: (meta: TelemSourceMeta) => Promise<TelemSourceMeta | null>;
}

class TelemRegistry {
  private readonly internal: Map<string, TelemSourceMeta>;

  constructor() {
    this.internal = new Map();
  }

  set(meta: TelemSourceMeta): void {
    this.internal.set(meta.key, meta);
  }

  get(key: string): TelemSourceMeta {
    const v = this.internal.get(key);
    if (v == null) throw new Error(`Telemetry source with key ${key} not found.`);
    return v;
  }

  delete(key: string): boolean {
    return this.internal.delete(key);
  }
}

export class TelemService implements TelemProvider {
  registry: TelemRegistry;
  factories: TelemFactory[];

  constructor(factories: TelemFactory[]) {
    this.factories = factories;
    this.registry = new TelemRegistry();
  }

  get<T>(key: string): T {
    return this.registry.get(key) as T;
  }

  async new(meta: TelemSourceMeta): Promise<TelemSourceMeta | null> {
    for (const factory of this.factories) {
      const source = await factory.new(meta);
      if (source != null) {
        this.registry.set(source);
        return source;
      }
    }
    return null;
  }
}
