import clsx from "clsx";
import { useRef, useState } from "react";
import { AiOutlineClose } from "react-icons/ai";
import Input from "../Input/Input";
import { List } from "../List";
import { useListContext } from "../List/ListContext";
import { Key, TypedListColumn, TypedListEntry } from "../List/Types";
import ListSearch from "../List/ListSearch";
import Space from "../Space/Space";
import { Tag } from "../Tag";
import "./SelectMultiple.css";
import Button from "../Button";
import useClickoutside from "../../Hooks/useClickOutside";
import { Theme } from "../../Theme";

export interface SelectMultipleProps<
  K extends Key,
  E extends TypedListEntry<K>
> {
  options?: E[];
  columns?: TypedListColumn<K, E>[];
  listPosition?: "top" | "bottom";
}

export default function SelectMultiple<
  K extends Key,
  E extends TypedListEntry<K>
>({
  options = [],
  columns = [],
  listPosition = "bottom",
}: SelectMultipleProps<K, E>) {
  const [visible, setVisible] = useState(false);
  const divRef = useRef<HTMLDivElement>(null);
  useClickoutside(divRef, () => setVisible(false));
  return (
    <List data={options}>
      <Space
        className="pluto-select-multiple__container"
        ref={divRef}
        empty
        reverse={listPosition === "top"}
      >
        <ListSearch
          Input={({ value, onChange }) => {
            return (
              <SelectMultipleInput
                tagKey={"name"}
                value={value}
                focused={visible}
                onFocus={() => setVisible(true)}
                onChange={onChange}
              />
            );
          }}
        />
        <Space
          className={clsx(
            "pluto-select-multiple__list",
            `pluto-select-multiple__list--${listPosition}`,
            `pluto-select-multiple__list--${visible ? "visible" : "hidden"}`
          )}
          empty
        >
          <List.Column.Header columns={columns} />
          <List.Core.Virtual itemHeight={30}>
            {(props) => <List.Column.Item {...props} />}
          </List.Core.Virtual>
        </Space>
      </Space>
    </List>
  );
}

interface SelectMultipleInputProps<K extends Key, E extends TypedListEntry<K>> {
  value?: string;
  onChange?: (value: string) => void;
  focused: boolean;
  onFocus: () => void;
  tagKey: keyof E;
}

const SelectMultipleInput = <K extends Key, E extends TypedListEntry<K>>({
  value,
  onChange,
  focused,
  onFocus,
  tagKey,
}: SelectMultipleInputProps<K, E>) => {
  const { selected, sourceData, onSelect, clearSelected } = useListContext<
    K,
    E
  >();

  const { theme } = Theme.useContext();
  return (
    <Space
      direction="horizontal"
      empty
      className="pluto-select-multiple__input__container"
      align="stretch"
      grow
    >
      <Input
        className="pluto-select-multiple__input__input"
        placeholder="Search"
        value={value}
        onChange={onChange}
        autoFocus={focused}
        onFocus={onFocus}
      />
      <Space
        direction="horizontal"
        className="pluto-select-multiple__input__tags"
        align="center"
        grow={6}
      >
        {selected
          .map((k) => sourceData.find((v) => v.key === k))
          .map((e, i) => {
            if (!e) return null;
            return (
              <Tag
                color={theme.colors.visualization.palettes.default[i]}
                onClose={() => onSelect(e.key)}
                size="small"
                variant="outlined"
              >
                {e[tagKey]}
              </Tag>
            );
          })}
      </Space>
      <Button.IconOnly
        className="pluto-select-multiple__input__tags__close"
        variant="outlined"
        onClick={clearSelected}
      >
        <AiOutlineClose />
      </Button.IconOnly>
    </Space>
  );
};
