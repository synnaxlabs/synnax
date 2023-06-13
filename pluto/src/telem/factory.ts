import { ModifiableTelemSourceMeta } from "@/telem/meta";

export interface TelemFactory {
  type: string;
  create: (key: string, type: string, props: any) => ModifiableTelemSourceMeta | null;
}

export class CompoundTelemFactory {
  factories: TelemFactory[];

  type = "compound";

  constructor(factories: TelemFactory[]) {
    this.factories = factories;
  }

  add(factory: TelemFactory): void {
    this.factories.push(factory);
  }

  change(factory: TelemFactory): void {
    this.remove(factory.type);
    this.add(factory);
  }

  remove(type: string): void {
    this.factories = this.factories.filter((f) => f.type !== type);
  }

  create(key: string, type: string, props: any): ModifiableTelemSourceMeta | null {
    for (const factory of this.factories) {
      const source = factory.create(key, type, props);
      if (source != null) return source;
    }
    return null;
  }
}
