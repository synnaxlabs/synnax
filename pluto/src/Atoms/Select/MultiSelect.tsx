import clsx from "clsx";
import { RefObject, useEffect, useRef, useState } from "react";
import { AiFillClockCircle, AiOutlineClose } from "react-icons/ai";
import { useThemeContext } from "../../Theme/ThemeContext";
import Button, { IconButton } from "../Button/Button";
import Input from "../Input/Input";
import { ColumnHeader, ColumnItem } from "../List/ColumnList";
import VirtualCore from "../List/CoreList";
import List from "../List/List";
import {
  Key,
  TypedColumn,
  TypedListEntry,
  useListContext,
} from "../List/ListContext";
import SearchList from "../List/SearchList";
import Space from "../Space/Space";
import Tag from "../Tag/Tag";
import IconText from "../Typography/IconText";
import "./MultiSelect.css";

export interface MultiSelectProps<K extends Key, E extends TypedListEntry<K>> {
  options: E[];
  columns: TypedColumn<K, E>[];
  listPosition: "top" | "bottom";
}

interface MultiSelectInputProps<K extends Key, E extends TypedListEntry<K>> {
  value: string;
  onChange: (value: string) => void;
  focused: boolean;
  onFocus: () => void;
  tagKey: keyof E;
}

const MultiSelectInput = <K extends Key, E extends TypedListEntry<K>>({
  value,
  onChange,
  focused,
  onFocus,
  tagKey,
}: MultiSelectInputProps<K, E>) => {
  const { selected, sourceData, onSelect, clearSelected } = useListContext<
    K,
    E
  >();

  const { theme } = useThemeContext();
  return (
    <Space
      direction="horizontal"
      empty
      className="pluto-multi-select__input__container"
    >
      <Input
        className="pluto-multi-select__input__input"
        placeholder="Search"
        value={value}
        onChange={onChange}
        autoFocus={focused}
        onFocus={onFocus}
      />
      <Space
        direction="horizontal"
        className="pluto-multi-select__input__tags"
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
        <IconButton
          className="pluto-multi-select__input__tags__close"
          variant="outlined"
          onClick={clearSelected}
        >
          <AiOutlineClose />
        </IconButton>
      </Space>
    </Space>
  );
};

export default function MultiSelect<
  K extends Key,
  E extends TypedListEntry<K>
>({ options, columns, listPosition = "top" }: MultiSelectProps<K, E>) {
  const [listVisible, setListVisible] = useState(false);
  const divRef = useRef<HTMLDivElement>(null);
  useClickoutside(divRef, () => setListVisible(false));

  return (
    <List data={options}>
      <Space
        className="pluto-multi-select__container"
        ref={divRef}
        empty
        reverse={listPosition === "top"}
      >
        <SearchList
          Input={({ value, onChange }) => {
            return (
              <MultiSelectInput
                tagKey={"name"}
                value={value}
                focused={listVisible}
                onFocus={() => setListVisible(true)}
                onChange={onChange}
              />
            );
          }}
        />
        <Space
          className={clsx(
            "pluto-multi-select__list",
            `pluto-multi-select__list--${listPosition}`
          )}
          style={{ opacity: listVisible ? 1 : 0, zIndex: listVisible ? 1 : -1 }}
          empty
        >
          <ColumnHeader columns={columns} />
          <VirtualCore itemHeight={30}>
            {(props) => <ColumnItem {...props} />}
          </VirtualCore>
        </Space>
      </Space>
    </List>
  );
}

function useClickoutside(ref: RefObject<any>, onClickOutside: () => void) {
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (ref.current && !ref.current.contains(event.target)) {
        onClickOutside();
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [ref, onClickOutside]);
}
