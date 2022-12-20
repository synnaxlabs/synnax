import { TimeSpan, TimeStamp } from "@synnaxlabs/client";
import { Text, List, Space } from "@synnaxlabs/pluto";

import { selectRange, useSelectRanges } from "../store";
import type { Range } from "../store";

export const RangesAccordionEntry = (): JSX.Element => {
  const ranges = useSelectRanges();
  return (
    <Space style={{ height: "100%" }} empty>
      <List data={ranges} onSelect={([key]) => selectRange(key)} selectMultiple={false}>
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
                      endTS.sub(startTS) < TimeSpan.Day ? "timeShort" : "dateTimeShort"
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
