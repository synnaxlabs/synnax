import { useState } from "react";

import { useAsyncEffect } from "@synnaxlabs/pluto";
import { OS } from "@synnaxlabs/x";

export interface SynnaxDownloadURLProps {
  os: OS;
}

const VERSION =
  "https://raw.githubusercontent.com/synnaxlabs/synnax/main/synnax/VERSION";

export const SynnaxDownloadURL = ({ os }: SynnaxDownloadURLProps): string => {
  const [url, setUrl] = useState("");
  useAsyncEffect(async () => {
    const version = await (await fetch(VERSION)).text();
    const url = `https://github.com/synnaxlabs/synnax/releases/download/synnax%2Fv0.4.0/synnax-${version}-${os.toLowerCase()}`;
    setUrl(url);
  }, [os]);
  return url;
};
