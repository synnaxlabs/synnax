// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { Align, Input, Select } from "@synnaxlabs/pluto";
import { useQuery } from "@tanstack/react-query";
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { type AxisKey, type XAxisKey, type YAxisKey } from "@/lineplot/axis";
import {
  SelectAxisInputItem,
  SelectMultipleAxesInputItem,
} from "@/lineplot/SelectAxis";
import { useSelect } from "@/lineplot/selectors";
import { setRanges, setXChannel, setYChannels } from "@/lineplot/slice";
import { Range } from "@/range";
import { useSelectMultiple } from "@/range/selectors";

export interface DataProps {
  layoutKey: string;
}

export const Data = ({ layoutKey }: DataProps): ReactElement | null => {
  const vis = useSelect(layoutKey);
  const dispatch = useDispatch();
  const allRanges = useSelectMultiple();

  const handleYChannelSelect = useCallback(
    (key: AxisKey, value: readonly channel.Key[]): void => {
      dispatch(
        setYChannels({
          key: layoutKey,
          axisKey: key as YAxisKey,
          channels: value as channel.Keys,
        }),
      );
    },
    [dispatch, layoutKey],
  );

  const handleXChannelSelect = useCallback(
    (key: AxisKey, value: channel.Key): void => {
      dispatch(
        setXChannel({
          key: layoutKey,
          axisKey: key as XAxisKey,
          channel: value,
        }),
      );
    },
    [dispatch, layoutKey],
  );

  const channels = useQuery({
    queryKey: ["channels"],
    queryFn: async () => {
        const result = await fetch("http://127.0.0.1:5000/api/v1/list", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
        });
        const json = await result.json();
        console.log(json);
        return json.channels.map((ch) => ({
          key: ch,
          name: ch,
        }))
    }
  })


  return (
    <Align.Space style={{ padding: "2rem", width: "100%" }}>
      <Input.Item direction="x" label="Y1" grow>
        <Select.Multiple<{key: string, name: string}>
          value={vis.channels.y1}
          grow
          columns={[
            {
              key: "name",
              name: "name",
            }
          ]}
          onChange={(v) => handleYChannelSelect("y1", v)}
          data={channels.data ?? []}
        />
      </Input.Item>
      {/* <SelectMultipleAxesInputItem
        axis={"y1"}
        onChange={handleYChannelSelect}
        value={vis.channels.y1}
        align="center"
        grow
        select={{ location: "top" }}
      />
      <SelectMultipleAxesInputItem
        axis={"y2"}
        onChange={handleYChannelSelect}
        value={vis.channels.y2}
        grow
        select={{ location: "top" }}
      />
      <SelectAxisInputItem
        axis={"x1"}
        onChange={handleXChannelSelect}
        value={vis.channels.x1}
        grow
        select={{ location: "top" }}
      /> */}
    </Align.Space>
  );
};
