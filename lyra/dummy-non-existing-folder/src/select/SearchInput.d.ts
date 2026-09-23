import { type Dialog } from "../dialog";
import { type Input } from "../input";
/** Props for {@link SearchInput}. */
export interface SearchInputProps {
    searchPlaceholder?: string;
    /** Called as the user types. Leave it unset to hide the input. */
    onSearch?: (term: string) => void;
    /** Buttons rendered inside the input, after the term. */
    actions?: Input.TextProps["children"];
    dialogVariant?: Dialog.FrameProps["variant"];
    loading?: boolean;
}
/** The search field at the top of a select {@link Dialog}. */
export declare const SearchInput: ({ searchPlaceholder, onSearch, actions, dialogVariant, loading, }: SearchInputProps) => import("react").JSX.Element;
//# sourceMappingURL=SearchInput.d.ts.map