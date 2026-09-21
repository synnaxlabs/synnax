// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect, useState } from "react";

import { type Client, getFromURL } from "@/components/client/Client";

export interface VarProps {
  py: string;
  ts: string;
}

export const Var = ({ py, ts }: VarProps) => {
  const [client, setClient] = useState<Client | null>(null);

  useEffect(() => {
    const updateFromURL = () => {
      const c = getFromURL();
      setClient(c);
    };
    updateFromURL();
    window.addEventListener("popstate", updateFromURL);
    window.addEventListener("urlchange", updateFromURL);
    return () => {
      window.removeEventListener("popstate", updateFromURL);
      window.removeEventListener("urlchange", updateFromURL);
    };
  }, []);

  const value = client === "typescript" ? ts : py;

  return <code>{value}</code>;
};
