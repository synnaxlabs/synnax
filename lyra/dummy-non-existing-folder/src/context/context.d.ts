import { type NotUndefined } from "@synnaxlabs/x";
export interface BaseCreateParams {
    displayName: string;
}
export interface CreateWithDefaultParams<T> extends BaseCreateParams {
    defaultValue: T;
}
export interface CreateWithoutDefaultParams extends BaseCreateParams {
    providerName: string;
}
export interface Creator {
    <T>(params: CreateWithDefaultParams<T>): [React.Context<T>, () => T];
    <T extends NotUndefined>(params: CreateWithoutDefaultParams): [React.Context<T | undefined>, (hookOrComponentName: string) => T];
}
/**
 * @param params.displayName - The display name of the context.
 * @param params.defaultValue - The default value of the context.
 * @param params.providerName - The name of the provider where the context is used.
 * @returns A tuple containing the context and the hook to use the context. If a default
 * value is not provided, the context will be of type `React.Context<T | undefined>`,
 * and the hook will throw an error if used outside of the context. If a default value
 * is provided, the context will be of type `React.Context<T>`, and the hook will return
 * the default value if used outside of the context.
 */
export declare const create: Creator;
//# sourceMappingURL=context.d.ts.map