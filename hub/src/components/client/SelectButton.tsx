// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Select } from "@synnaxlabs/pluto";
import { useEffect, useState } from "react";

import { type Client, CLIENTS, getFromURL, setInURL } from "@/components/client/Client";

const indexMap = new Map<Client, number>();
CLIENTS.forEach((c, i) => indexMap.set(c.key, i));

export interface SelectButtonProps {
  clients: Client[];
}

export const SelectButton = ({ clients }: SelectButtonProps) => {
  const [client, setClient] = useState<Client>(clients[0]);

  // Map the clients so the order of the clients is consistent between the props and
  // the data passed to the select button.
  const data = clients.map((c) => CLIENTS[indexMap.get(c) as number]);

  useEffect(() => {
    const updateFromURL = () => {
      const c = getFromURL();
      if (c) setClient(c);
    };
    updateFromURL();
    window.addEventListener("popstate", updateFromURL);
    window.addEventListener("urlchange", updateFromURL);
    return () => {
      window.removeEventListener("popstate", updateFromURL);
      window.removeEventListener("urlchange", updateFromURL);
    };
  }, []);

  const handleChange = (c: Client) => {
    setInURL(c);
    setClient(c);
  };

  return (
    <Select.Static
      className="styled-scrollbar"
      location="bottom"
      resourceName="client"
      data={data}
      value={client}
      allowNone={false}
      onChange={handleChange}
      virtual={false}
    />
  );
};
