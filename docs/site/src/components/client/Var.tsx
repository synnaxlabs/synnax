// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect, useState } from "react";

import { type Client, getFromURL } from "@/components/client/client";

export interface VarProps {
  py: string;
  ts: string;
}

export const Var = ({ py, ts }: VarProps) => {
  const [client, setClient] = useState<Client | null>(null);

  useEffect(() => {
    const update = () => {
      const c = getFromURL();
      setClient(c);
    };
    update();
    const i = setInterval(update, 200);
    return () => clearInterval(i);
  }, []);

  const value = client === "typescript" ? ts : py;

  return <code>{value}</code>;
};
