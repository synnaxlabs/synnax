import "@/menu/Item.css";

import { type FunctionComponent, type ReactElement } from "react";

import { Align } from "@/align";
import { Button } from "@/button";
import { CSS } from "@/css";
import { useMenuContext } from "@/menu/Menu";
import { Text as TriggersText } from "@/triggers/Text";
import { type Trigger } from "@/triggers/triggers";

export interface MenuItemExtraProps {
  itemKey: string;
  trigger?: Trigger;
}

export interface ItemProps extends Button.ButtonProps, MenuItemExtraProps {}

export const CoreItem: FunctionComponent<ItemProps> = (props): ReactElement => {
  const { itemKey, trigger, className, onClick, size, ...rest } = props;

  const { onClick: ctxOnClick, selected, level = "p", iconSpacing } = useMenuContext();

  const handleClick: Button.ButtonProps["onClick"] = (e) => {
    ctxOnClick(itemKey);
    onClick?.(e);
  };

  const _selected = selected === itemKey;

  return (
    <Button.Button
      level={level}
      {...rest}
      noWrap={true}
      onClick={handleClick}
      variant="text"
      className={CSS(CSS.B("menu-item"), CSS.selected(_selected), className)}
      size={size ?? iconSpacing}
      endIcon={
        trigger && (
          <Align.Space
            className={CSS(CSS.BE("menu-item", "trigger"))}
            direction="x"
            size={0.5}
          >
            <TriggersText level={level} trigger={trigger} />
          </Align.Space>
        )
      }
    />
  );
};

export interface ItemIconProps extends Button.IconProps, MenuItemExtraProps {}

export const ItemIcon: FunctionComponent<ItemIconProps> = (props): ReactElement => {
  const { itemKey, trigger, className, onClick, size, ...rest } = props;

  const { onClick: ctxOnClick, selected, iconSpacing } = useMenuContext();

  const handleClick: Button.ButtonProps["onClick"] = (e) => {
    ctxOnClick(itemKey);
    onClick?.(e);
  };

  const _selected = selected === itemKey;

  return (
    <Button.Icon
      {...rest}
      onClick={handleClick}
      variant="text"
      className={CSS(CSS.B("menu-item"), CSS.selected(_selected), className)}
      size={size ?? iconSpacing}
    />
  );
};

export interface ItemLinkProps extends Button.LinkProps, MenuItemExtraProps {}

export const ItemLink: FunctionComponent<ItemLinkProps> = (props): ReactElement => {
  const { itemKey, trigger, className, onClick, size, ...rest } = props;

  const { onClick: ctxOnClick, selected, level = "p", iconSpacing } = useMenuContext();

  const handleClick: Button.ButtonProps["onClick"] = (e) => {
    ctxOnClick(itemKey);
    onClick?.(e);
  };

  const _selected = selected === itemKey;

  return (
    <Button.Link
      level={level}
      {...rest}
      noWrap={true}
      onClick={handleClick}
      variant="text"
      className={CSS(CSS.B("menu-item"), CSS.selected(_selected), className)}
      size={size ?? iconSpacing}
      endIcon={
        trigger && (
          <Align.Space
            className={CSS(CSS.BE("menu-item", "trigger"))}
            direction="x"
            size={0.5}
          >
            <TriggersText level={level} trigger={trigger} />
          </Align.Space>
        )
      }
    />
  );
};

type CoreItemType = typeof CoreItem;

export interface ItemType extends CoreItemType {
  Icon: typeof ItemIcon;
  Link: typeof ItemLink;
}

/**
 * Menu.Item renders a menu item.
 *
 * @param props - Props for the component. Identical props to those of Use except
 * for the ones listed below.
 * @param props.itemKey - The key of the item. This is used to identify the item and
 * is passed to the onChange callback of the Menu.
 */
export const Item = CoreItem as ItemType;
Item.Icon = ItemIcon;
Item.Link = ItemLink;
