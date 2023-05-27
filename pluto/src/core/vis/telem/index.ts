import { Bound, LazyArray, ZERO_BOUND } from "@synnaxlabs/x";

export interface TelemMeta {
  key: string;
  type: string;
}

export interface XYTelemMeta {
  key: string;
  type: "xy";
}

export interface DynamicXYTelemMeta {
  key: string;
  type: "dynamic-xy";
}

export interface XYTelem extends TelemMeta {
  x: () => Promise<LazyArray[]>;
  y: () => Promise<LazyArray[]>;
  xBound: () => Promise<Bound>;
  yBound: () => Promise<Bound>;
}

export interface DynamicXYTelem extends XYTelem {
  onChange: (f: () => void) => void;
}

class EmptyStaticXYTelem implements XYTelem {
  key = "empty";
  type = "empty";

  async x(): Promise<LazyArray[]> {
    return [];
  }

  async y(): Promise<LazyArray[]> {
    return [];
  }

  async xBound(): Promise<Bound> {
    return ZERO_BOUND;
  }

  async yBound(): Promise<Bound> {
    return ZERO_BOUND;
  }
}

export const ZERO_XY_TELEM = new EmptyStaticXYTelem();

export type TelemProvider = <T extends TelemMeta>(key: string) => T;

export class TelemService {
  telem: Map<string, TelemMeta>;

  constructor() {
    this.telem = new Map();
  }

  register(meta: TelemMeta): void {
    this.telem.set(meta.key, meta);
  }

  get(key: string): TelemMeta | undefined {
    return this.telem.get(key);
  }

  provider(): TelemProvider {
    return <T extends TelemMeta>(key: string): T => {
      const meta = this.get(key);
      if (meta == null) throw new Error(`No telem with key ${key}`);
      return meta as T;
    };
  }
}
