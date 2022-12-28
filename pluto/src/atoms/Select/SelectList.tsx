import { List } from "../List";
import { ListColumnHeaderProps } from "../List/ListColumn";
import { ListSelectorProps } from "../List/ListSelector";

import { RenderableRecord } from "@/util/record";

export interface SelectListProps<E extends RenderableRecord<E>>
  extends ListSelectorProps<E>,
    ListColumnHeaderProps<E> {}

export const SelectList = <E extends RenderableRecord>({
  value,
  onChange,
  allowMultiple,
  ...props
}: SelectListProps<E>): JSX.Element => (
  <>
    <List.Selector value={value} onChange={onChange} allowMultiple={allowMultiple} />
    <List.Column.Header {...props} />
    <List.Core.Virtual itemHeight={List.Column.itemHeight}>
      {List.Column.Item}
    </List.Core.Virtual>
  </>
);
