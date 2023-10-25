const VERSION =
    "https://raw.githubusercontent.com/synnaxlabs/synnax/main/synnax/pkg/version/VERSION";

export const getVersion = async (): string => 
   await (await fetch(VERSION)).text();