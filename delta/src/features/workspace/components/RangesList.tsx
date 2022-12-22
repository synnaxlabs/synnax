import { TimeSpan, TimeStamp } from "@synnaxlabs/client";
import { Text, List, Space } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { selectRange, useSelectRanges, useSelectSelectedRange } from "../store";
import type { Range } from "../store";

export const RangesAccordionEntry = (): JSX.Element => {
  const ranges = useSelectRanges();
  const selectedRange = useSelectSelectedRange();
  const dispatch = useDispatch();
  return (
    <Space style={{ height: "100%" }} empty>
      <List
        data={ranges}
        onSelect={([key]) => dispatch(selectRange(key ?? null))}
        selected={selectedRange == null ? [] : [selectedRange.key]}
        selectMultiple={false}
      >
        <List.Column.Header<Range>
          columns={[
            {
              key: "name",
              label: "Name",
            },
            {
              key: "start",
              label: "Start",
              render: ({ entry: { start }, style }) => {
                return (
                  <Text.DateTime level="p" style={style}>
                    {new TimeStamp(start).date()}
                  </Text.DateTime>
                );
              },
            },
            {
              key: "end",
              label: "End",
              render: ({ entry: { start, end }, style }) => {
                const startTS = new TimeStamp(start);
                const endTS = new TimeStamp(end);
                return (
                  <Text.DateTime
                    level="p"
                    style={style}
                    format={
                      endTS.span(startTS) < TimeSpan.Day ? "timeShort" : "dateTimeShort"
                    }
                  >
                    {endTS}
                  </Text.DateTime>
                );
              },
            },
          ]}
        />
        <List.Core.Virtual itemHeight={30} style={{ height: "100%" }}>
          {(props) => <List.Column.Item {...props} />}
        </List.Core.Virtual>
      </List>
    </Space>
  );
};
