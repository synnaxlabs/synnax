// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/DateTime.css";

import { type record, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import compromise from "compromise";
import compromiseDates, { type DatesMethods } from "compromise-dates";
import { type CSSProperties, type FC, type ReactElement, useState } from "react";

import { Button } from "@/button";
import { renderProp } from "@/component/renderProp";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Numeric } from "@/input/Numeric";
import { Text as InputText, type TextProps } from "@/input/Text";
import { type Control } from "@/input/types";
import { List } from "@/list";
import { Nav } from "@/nav";
import { Select } from "@/select";
import { Text as TelemText } from "@/telem/text";
import { Text } from "@/text";
import { Triggers } from "@/triggers";

export interface DateTimeProps
  extends Omit<TextProps, "type" | "value" | "onChange">, Control<number> {}

export const DateTime = ({
  value,
  onChange,
  onBlur,
  onlyChangeOnBlur,
  variant,
  ...rest
}: DateTimeProps): ReactElement => {
  const [tempValue, setTempValue] = useState<string | null>(null);

  const handleChange = (next: string | number, override: boolean = false): void => {
    const nextStr = next.toString();
    setTempValue(nextStr);

    const isString = typeof next === "string";
    if (!isString) {
      onChange(Number(next));
      setTempValue(null);
      return;
    }

    const nextTS = new TimeStamp(nextStr, "local");

    if (!onlyChangeOnBlur || override) {
      onChange(Number(nextTS.valueOf()));
      setTempValue(null);
    }
  };

  const handleBlur: React.FocusEventHandler<HTMLInputElement> = (e) => {
    handleChange(e.target.value, true);
    setTempValue(null);
    onBlur?.(e);
  };

  const tsValue = new TimeStamp(value, "UTC");
  const parsedValue = tsValue.toString("ISO", "local").slice(0, -1);

  const [visible, setVisible] = useState(false);

  return (
    <Dialog.Frame
      visible={visible}
      variant="modal"
      zIndex={500}
      onVisibleChange={setVisible}
    >
      <InputText
        className={CSS.BE("input", "datetime")}
        variant={variant}
        type="datetime-local"
        onBlur={handleBlur}
        required={false}
        value={tempValue ?? parsedValue}
        onChange={handleChange}
        step={0.00001}
        {...rest}
      >
        <Button.Button onClick={() => setVisible(!visible)} variant={variant}>
          <Icon.Calendar />
        </Button.Button>
      </InputText>
      <DateTimeModal
        value={tsValue}
        onChange={(next) => onChange(Number(next.valueOf()))}
      />
    </Dialog.Frame>
  );
};

const nlp = compromise.extend(compromiseDates);

interface DateTimeModalProps {
  value: TimeStamp;
  onChange: (next: TimeStamp) => void;
}

const SAVE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

const DateTimeModal = ({ value, onChange }: DateTimeModalProps): ReactElement => {
  const { close } = Dialog.useContext();
  return (
    <Dialog.Dialog>
      <Flex.Box className={CSS.B("datetime-modal")} empty>
        <Flex.Box className={CSS.B("datetime-modal-container")}>
          <Flex.Box x className={CSS.B("header")}>
            <TelemText.TimeStamp level="h3" format="preciseDate">
              {value}
            </TelemText.TimeStamp>
          </Flex.Box>
          <Button.Button variant="text" className={CSS.B("close-btn")} onClick={close}>
            <Icon.Close />
          </Button.Button>
          <Flex.Box x className={CSS.B("content")}>
            <AISelector value={value} onChange={onChange} close={close} />
            <Calendar value={value} onChange={onChange} />
          </Flex.Box>
        </Flex.Box>
        <Nav.Bar location="bottom" size="7rem">
          <Nav.Bar.Start gap="small">
            <Triggers.Text level="small" trigger={SAVE_TRIGGER} />
            <Text.Text level="small">To Finish</Text.Text>
          </Nav.Bar.Start>
          <Nav.Bar.End>
            <Button.Button onClick={close} variant="outlined">
              Done
            </Button.Button>
          </Nav.Bar.End>
        </Nav.Bar>
      </Flex.Box>
    </Dialog.Dialog>
  );
};

interface AISuggestion {
  key: string;
  name: string;
  onSelect: () => void;
}

const AIListItem = (props: List.ItemRenderProps<string>): ReactElement => {
  const item = List.useItem<string, AISuggestion>(props.key);
  return (
    <List.Item {...props}>
      <Text.Text>{item?.name}</Text.Text>
    </List.Item>
  );
};

const aiListItem = renderProp(AIListItem);

interface AISelectorProps {
  value: TimeStamp;
  close: () => void;
  onChange: (next: TimeStamp) => void;
}

const AISelector = ({
  value: pValue,
  onChange,
  close,
}: AISelectorProps): ReactElement => {
  const [value, setValue] = useState<string>("");
  const [entries, setEntries] = useState<AISuggestion[]>([]);
  const { data, getItem } = List.useStaticData<string>({ data: entries });

  const handleChange = (next: string): void => {
    const processed = nlp(next) as DatesMethods;
    const dates = processed.dates().get() as DateInfo[];
    const entries: AISuggestion[] = [];
    entries.push(
      ...dates.map((d) => {
        const nextTS = new TimeStamp(d.start);
        return {
          key: d.start,
          name: nextTS.toString("preciseDate", "local"),
          onSelect: () => {
            onChange(nextTS);
            close();
          },
        };
      }),
    );
    setEntries(entries);
    const durations = (processed.durations() as any).get() as DurationInfo[];
    entries.push(
      ...durations.map((d) => {
        let span = new TimeSpan(0);
        if (d.hour != null) span = span.add(TimeSpan.hours(d.hour));
        if (d.minute != null) span = span.add(TimeSpan.minutes(d.minute));
        if (d.second != null) span = span.add(TimeSpan.seconds(d.second));
        if (d.millisecond != null)
          span = span.add(TimeSpan.milliseconds(d.millisecond));
        const next = pValue.add(span);
        return {
          key: next.valueOf().toString(),
          name: next.toString("preciseDate", "local"),
          onSelect: () => {
            onChange(next);
            close();
          },
        };
      }),
    );
    setValue(next);
  };
  const handleSelect = (key: string | null): void => {
    const entry = entries.find((e) => e.key === key);
    if (entry) entry.onSelect();
    setValue("");
    setEntries([]);
  };
  return (
    <Flex.Box pack y className={CSS.B("ai-selector")} background={1} full="y">
      <InputText
        value={value}
        onChange={handleChange}
        autoFocus
        placeholder="AI Suggestion"
        full="x"
      />
      <Select.Frame data={data} allowNone onChange={handleSelect} getItem={getItem}>
        <List.Items<string, AISuggestion>
          className={CSS.B("ai-list")}
          bordered
          borderColor={5}
          full="y"
          emptyContent={
            <Flex.Box empty grow align="center" justify="center">
              <Flex.Box y gap="tiny">
                <Text.Text level="small" color="var(--pluto-gray-l7)">
                  "April 1 at 2PM"
                </Text.Text>
                <Text.Text level="small" color="var(--pluto-gray-l7)">
                  "Add 2 two hours"
                </Text.Text>
                <Text.Text level="small" color="var(--pluto-gray-l7)">
                  "Next Friday"
                </Text.Text>
              </Flex.Box>
            </Flex.Box>
          }
        >
          {aiListItem}
        </List.Items>
      </Select.Frame>
    </Flex.Box>
  );
};

interface Month {
  name: string;
  days: number;
}

const MONTHS: Month[] = [
  { name: "January", days: 31 },
  { name: "February", days: 28 },
  { name: "March", days: 31 },
  { name: "April", days: 30 },
  { name: "May", days: 31 },
  { name: "June", days: 30 },
  { name: "July", days: 31 },
  { name: "August", days: 31 },
  { name: "September", days: 30 },
  { name: "October", days: 31 },
  { name: "November", days: 30 },
  { name: "December", days: 31 },
];

interface DateInfo {
  start: string;
}

interface DurationInfo {
  hour?: number;
  minute?: number;
  second?: number;
  millisecond?: number;
}

interface CalendarProps {
  value: TimeStamp;
  onChange: (next: TimeStamp) => void;
}

const Calendar = ({ value, onChange }: CalendarProps): ReactElement => {
  const month = value.month;
  const year = value.year;
  const day = value.day;

  const handleMonthChange = (next: number): void => onChange(value.setMonth(next));

  const handleYearChange = (next: number): void => onChange(value.setYear(next));

  const handleDayChange = (next: number): void => onChange(value.setDay(next));

  return (
    <Flex.Box pack x className={CSS.B("datetime-picker")} rounded>
      <Flex.Box pack y align="stretch" className={CSS.B("calendar")}>
        <Flex.Box pack x grow className={CSS.B("calendar-header")}>
          <Button.Button
            onClick={() => handleMonthChange(month - 1)}
            variant="outlined"
          >
            <Icon.Caret.Left />
          </Button.Button>
          <Text.Text
            level="small"
            style={{ flexGrow: 1, paddingLeft: "1rem" }}
            className={CSS.BE("calendar-header", "month")}
          >
            {MONTHS[month].name}
          </Text.Text>
          <Button.Button
            onClick={() => handleMonthChange(month + 1)}
            style={{ borderTopRightRadius: 0 }}
            variant="outlined"
          >
            <Icon.Caret.Right />
          </Button.Button>
        </Flex.Box>
        <Flex.Box pack x grow sharp>
          <Button.Button onClick={() => handleYearChange(year - 1)} variant="outlined">
            <Icon.Caret.Left />
          </Button.Button>
          <Text.Text level="small" style={{ flexGrow: 1, paddingLeft: "1rem" }}>
            {year}
          </Text.Text>
          <Button.Button onClick={() => handleYearChange(year + 1)} variant="outlined">
            <Icon.Caret.Right />
          </Button.Button>
        </Flex.Box>
        <Flex.Box x wrap gap="tiny" style={{ padding: "0.5rem", height: "100%" }}>
          {Array.from({ length: MONTHS[month].days }).map((_, i) => (
            <Button.Button
              key={i}
              variant={i + 1 === day ? "outlined" : "text"}
              onClick={() => handleDayChange(i + 1)}
              square
            >
              <Text.Text level="small">{i + 1}</Text.Text>
            </Button.Button>
          ))}
        </Flex.Box>
      </Flex.Box>
      <TimeSelector value={value} onChange={onChange} />
    </Flex.Box>
  );
};

const TIME_LIST_ITEM_STYLE: CSSProperties = {
  padding: "0rem",
  paddingLeft: "2rem",
  height: "4rem",
  minHeight: "4rem",
  maxHeight: "4rem",
};

const TimeListItem = (props: List.ItemRenderProps<number>): ReactElement => (
  <Select.ListItem {...props} style={TIME_LIST_ITEM_STYLE}>
    {props.index}
  </Select.ListItem>
);

interface TimeListProps {
  value: number;
  onChange: (next: number) => void;
}

const timeListItem = renderProp(TimeListItem);

const createTimeList = (count: number): FC<TimeListProps> => {
  const data = Array.from({ length: count }, (_, i) => i);

  const TimeList = ({ value, onChange }: TimeListProps): ReactElement => (
    <Select.Frame<number, record.KeyedNamed<number>>
      data={data}
      value={value}
      onChange={onChange}
    >
      <List.Items<number, record.KeyedNamed<number>> className={CSS.B("time-list")}>
        {timeListItem}
      </List.Items>
    </Select.Frame>
  );
  TimeList.displayName = "TimeList";
  return TimeList;
};

const HoursList = createTimeList(24);
const MinutesList = createTimeList(60);
const SecondsList = createTimeList(60);

interface TimeSelectorProps {
  value: TimeStamp;
  onChange: (next: TimeStamp) => void;
}

const TimeSelector = ({ value, onChange }: TimeSelectorProps): ReactElement => (
  <Flex.Box pack y className={CSS.B("time-selector")}>
    <Flex.Box pack x grow className={CSS.B("time-selector-list")}>
      <HoursList
        value={value.hour}
        onChange={(next) => onChange(value.setHour(next))}
      />
      <MinutesList
        value={value.minute}
        onChange={(next) => onChange(value.setMinute(next))}
      />
      <SecondsList
        value={value.second}
        onChange={(next) => onChange(value.setSecond(next))}
      />
    </Flex.Box>
    <Numeric
      size="small"
      value={value.millisecond}
      onChange={(next) => onChange(value.setMillisecond(next))}
      endContent="ms"
      showDragHandle={false}
      borderColor={5}
    />
  </Flex.Box>
);
