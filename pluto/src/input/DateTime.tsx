// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/DateTime.css";

import { Icon } from "@synnaxlabs/media";
import { type KeyedNamed, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import compromise from "compromise";
import compromiseDates, { type DatesMethods } from "compromise-dates";
import { type FC, type ReactElement, useState } from "react";

import { Align } from "@/align";
import { Button } from "@/button";
import { CSS } from "@/css";
import { Dropdown } from "@/dropdown";
import { Input } from "@/input";
import { Text as InputText } from "@/input/Text";
import { type BaseProps } from "@/input/types";
import { List } from "@/list";
import { Nav } from "@/nav";
import { Text } from "@/text";
import { Triggers } from "@/triggers";
import { componentRenderProp } from "@/util/renderProp";

const applyTimezoneOffset = (ts: TimeStamp): TimeStamp =>
  ts.add(
    BigInt(TimeStamp.now().date().getTimezoneOffset() - ts.date().getTimezoneOffset()) *
      TimeSpan.MINUTE.valueOf(),
  );

export interface DateTimeProps extends BaseProps<number> {}

export const DateTime = ({
  value,
  onChange,
  onBlur,
  onlyChangeOnBlur,
  variant,
  ...props
}: DateTimeProps): ReactElement => {
  const [tempValue, setTempValue] = useState<string | null>(null);

  const handleChange = (next: string | number, override: boolean = false): void => {
    let nextStr = next.toString();
    setTempValue(nextStr);

    let nextTS = new TimeStamp(next, "UTC");
    if (nextStr.length < 23) nextStr += ".000";

    nextTS = applyTimezoneOffset(nextTS);
    let ok = false;
    try {
      const str = nextTS.fString("ISO", "local");
      ok = str.slice(0, -1) === nextStr;
    } catch (e) {
      console.error(e);
    }
    if (ok && !onlyChangeOnBlur) {
      onChange(Number(nextTS.valueOf()));
      setTempValue(null);
    }
    if (override) {
      if (ok) onChange(Number(nextTS.valueOf()));
      setTempValue(null);
    }
  };

  const handleBlur: React.FocusEventHandler<HTMLInputElement> = (e) => {
    handleChange(e.target.value, true);
    setTempValue(null);
    onBlur?.(e);
  };

  const tsValue = new TimeStamp(value, "UTC");
  const parsedValue = tsValue.fString("ISO", "local").slice(0, -1);

  const dProps = Dropdown.use();

  return (
    <Dropdown.Dialog {...dProps} variant="modal" zIndex={500} keepMounted={false}>
      <InputText
        className={CSS.BE("input", "datetime")}
        variant={variant}
        type="datetime-local"
        onBlur={handleBlur}
        required={false}
        value={tempValue ?? parsedValue}
        onChange={handleChange}
        step={0.00001}
        {...props}
      >
        <Button.Icon
          onClick={dProps.toggle}
          variant={variant === "natural" ? "text" : "outlined"}
        >
          <Icon.Calendar />
        </Button.Icon>
      </InputText>
      <DateTimeModal
        value={tsValue}
        onChange={(next) => onChange(Number(next.valueOf()))}
        close={dProps.close}
      />
    </Dropdown.Dialog>
  );
};

const nlp = compromise.extend(compromiseDates);

interface DateTimeModalProps {
  value: TimeStamp;
  onChange: (next: TimeStamp) => void;
  close: () => void;
}

const SAVE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

const DateTimeModal = ({
  value,
  onChange,
  close,
}: DateTimeModalProps): ReactElement => (
  <Align.Space className={CSS.B("datetime-modal")} empty>
    <Align.Space className={CSS.B("content")}>
      <Align.Space direction="x" className={CSS.B("header")}>
        <Text.DateTime level="h3" format="preciseDate">
          {value}
        </Text.DateTime>
      </Align.Space>
      <Button.Icon variant="text" className={CSS.B("close-btn")} onClick={close}>
        <Icon.Close />
      </Button.Icon>
      <Align.Space direction="x" className={CSS.B("content")}>
        <AISelector value={value} onChange={onChange} close={close} />
        <Calendar value={value} onChange={onChange} />
      </Align.Space>
    </Align.Space>
    <Nav.Bar location="bottom" size="7rem">
      <Nav.Bar.Start size="small">
        <Triggers.Text shade={7} level="small" trigger={SAVE_TRIGGER} />
        <Text.Text shade={7} level="small">
          To Finish
        </Text.Text>
      </Nav.Bar.Start>
      <Nav.Bar.End>
        <Button.Button onClick={close} variant="outlined">
          Done
        </Button.Button>
      </Nav.Bar.End>
    </Nav.Bar>
  </Align.Space>
);

interface AISuggestion {
  key: string;
  name: string;
  onSelect: () => void;
}

const AIListItem = (props: List.ItemProps<string, AISuggestion>): ReactElement => (
  <List.ItemFrame {...props}>
    <Text.Text level="p">{props.entry.name}</Text.Text>
  </List.ItemFrame>
);

const aiListItem = componentRenderProp(AIListItem);

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

  const handleChange = (next: string): void => {
    const processed = nlp(next) as DatesMethods;
    const dates = processed.dates().get() as DateInfo[];
    const entries: AISuggestion[] = [];
    entries.push(
      ...dates.map((d) => {
        const nextTS = applyTimezoneOffset(new TimeStamp(d.start, "UTC"));
        return {
          key: d.start,
          name: nextTS.fString("preciseDate", "local"),
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
        const next = applyTimezoneOffset(pValue.add(span));
        return {
          key: next.valueOf().toString(),
          name: next.fString("preciseDate", "local"),
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
    <Align.Pack direction="y" className={CSS.B("ai-selector")}>
      <Input.Text
        value={value}
        onChange={handleChange}
        autoFocus
        placeholder="AI Suggestion"
      />
      <List.List
        data={entries}
        emptyContent={
          <Align.Center empty grow>
            <Align.Space direction="y" size={0.5}>
              <Text.Text level="small" color="var(--pluto-gray-l5)">
                "April 1 at 2PM"
              </Text.Text>
              <Text.Text level="small" color="var(--pluto-gray-l5)">
                "Add 2 two hours"
              </Text.Text>
              <Text.Text level="small" color="var(--pluto-gray-l5)">
                "Next Friday"
              </Text.Text>
            </Align.Space>
          </Align.Center>
        }
      >
        <List.Selector<string, AISuggestion>
          value={null}
          allowMultiple={false}
          allowNone
          onChange={handleSelect}
        >
          <List.Hover initialHover={0}>
            <List.Core grow>{aiListItem}</List.Core>
          </List.Hover>
        </List.Selector>
      </List.List>
    </Align.Pack>
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

export const Calendar = ({ value, onChange }: CalendarProps): ReactElement => {
  const month = value.month;
  const year = value.year;
  const day = value.day;

  const handleMonthChange = (next: number): void => onChange(value.setMonth(next));

  const handleYearChange = (next: number): void => onChange(value.setYear(next));

  const handleDayChange = (next: number): void => onChange(value.setDay(next));

  return (
    <Align.Pack direction="x" className={CSS.B("datetime-picker")}>
      <Align.Pack
        direction="y"
        align="stretch"
        style={{ width: "37rem", height: "37rem" }}
        className={CSS.B("calendar")}
      >
        <Align.Pack direction="x" grow>
          <Button.Icon onClick={() => handleMonthChange(month - 1)} variant="outlined">
            <Icon.Caret.Left />
          </Button.Icon>
          <Text.WithIcon level="small" style={{ flexGrow: 1, paddingLeft: "1rem" }}>
            {MONTHS[month].name}
          </Text.WithIcon>
          <Button.Icon
            onClick={() => handleMonthChange(month + 1)}
            style={{ borderTopRightRadius: 0 }}
            variant="outlined"
          >
            <Icon.Caret.Right />
          </Button.Icon>
        </Align.Pack>
        <Align.Pack direction="x" grow>
          <Button.Icon onClick={() => handleYearChange(year - 1)} variant="outlined">
            <Icon.Caret.Left />
          </Button.Icon>
          <Text.WithIcon level="small" style={{ flexGrow: 1, paddingLeft: "1rem" }}>
            {year}
          </Text.WithIcon>
          <Button.Icon onClick={() => handleYearChange(year + 1)} variant="outlined">
            <Icon.Caret.Right />
          </Button.Icon>
        </Align.Pack>
        <Align.Space
          direction="x"
          wrap
          size={0.5}
          style={{ padding: "0.5rem", height: "100%" }}
        >
          {Array.from({ length: MONTHS[month].days }).map((_, i) => (
            <Button.Icon
              key={i}
              variant={i + 1 === day ? "outlined" : "text"}
              onClick={() => handleDayChange(i + 1)}
            >
              <Text.Text level="small">{i + 1}</Text.Text>
            </Button.Icon>
          ))}
        </Align.Space>
      </Align.Pack>
      <TimeSelector value={value} onChange={onChange} />
    </Align.Pack>
  );
};

const TimeListItem = (props: List.ItemProps<string, KeyedNamed>): ReactElement => {
  const { entry } = props;
  return (
    <List.ItemFrame {...props} style={{ padding: "0.5rem", paddingLeft: "2rem" }}>
      <Text.Text level="small">{entry.name}</Text.Text>
    </List.ItemFrame>
  );
};

interface TimeListProps {
  value: number;
  onChange: (next: number) => void;
}

const timeListItem = componentRenderProp(TimeListItem);

export const createTimeList = (count: number): FC<TimeListProps> => {
  const data = Array.from({ length: count }, (_, i) => ({
    key: i.toString(),
    name: i.toString(),
  }));
  const TimeList = ({ value, onChange }: TimeListProps): ReactElement => (
    <List.List<string, KeyedNamed> data={data}>
      <List.Selector<string, KeyedNamed>
        value={value.toString()}
        onChange={(next: string) => onChange(Number(next))}
        allowMultiple={false}
        allowNone={false}
      >
        <List.Core<string, KeyedNamed> className={CSS.B("time-list")}>
          {timeListItem}
        </List.Core>
      </List.Selector>
    </List.List>
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

export const TimeSelector = ({ value, onChange }: TimeSelectorProps): ReactElement => (
  <Align.Pack
    direction="y"
    className={CSS.B("time-selector")}
    style={{ height: "37rem" }}
  >
    <Align.Pack direction="x" grow>
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
    </Align.Pack>
    <Input.Numeric
      size="small"
      value={value.millisecond}
      onChange={(next) => onChange(value.setMillisecond(next))}
      endContent="ms"
      showDragHandle={false}
    />
  </Align.Pack>
);
