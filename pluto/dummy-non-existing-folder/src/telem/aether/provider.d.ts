import { z } from "zod";
import { aether } from "../../aether/aether";
import { type CompoundFactory, type Factory } from "./factory";
import { type Client } from "./remote";
export type ProviderState = z.input<typeof providerStateZ>;
export declare const providerStateZ: z.ZodObject<{}, z.core.$strip>;
export declare const PROVIDER_TYPE = "telem.Provider";
export declare const createProvider: (createFactory: (client: Client | null) => CompoundFactory) => aether.ComponentConstructor;
export declare const Provider: aether.ComponentConstructor;
export declare const REGISTRY: aether.ComponentRegistry;
export type FactoryConstructor = (client: Client | null) => Factory;
export declare const createRegistry: (...factoryConstructors: FactoryConstructor[]) => aether.ComponentRegistry;
//# sourceMappingURL=provider.d.ts.map