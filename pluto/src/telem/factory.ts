import { ModifiableTelemSourceMeta } from "@/telem/meta";

export interface TelemFactory {
  create: (key: string, type: string, props: any) => ModifiableTelemSourceMeta | null;
}

export class CompoundTelemFactory {
  factories: TelemFactory[];

  constructor(factories: TelemFactory[]) {
    this.factories = factories;
  }

  create(key: string, type: string, props: any): ModifiableTelemSourceMeta | null {
    for (const factory of this.factories) {
      const source = factory.create(key, type, props);
      if (source != null) return source;
    }
    return null;
  }
}
