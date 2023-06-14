import { ClientProvider, useClient } from "@/client/Context";
export type { ClientProviderProps } from "@/client/Context";

export const Client = {
  Provider: ClientProvider,
  use: useClient,
};
