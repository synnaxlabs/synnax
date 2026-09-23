import "./Swatch.css";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { type ReactElement } from "react";
import { type BaseSwatchProps } from "./BaseSwatch";
import { type PickerProps } from "./Picker";
export interface SwatchProps extends BaseSwatchProps, Pick<Dialog.FrameProps, "visible" | "onVisibleChange" | "initialVisible">, Pick<PickerProps, "onDelete" | "position"> {
    allowChange?: boolean;
    onlyChangeOnBlur?: boolean;
}
/**
 * A color swatch that opens a picker when clicked.
 * @param props - The props for the swatch. Unlisted props are passed to the underlying
 * button.
 * @param props.onChange - A function to call when the color changes.
 * @param props.onlyChangeOnBlur - If true, the swatch holds picker changes locally and
 * calls `onChange` once, when the picker closes. Set it where a live preview is not
 * worth a change per pixel of drag through the gradient. A change pending when the
 * swatch unmounts is dropped.
 */
export declare const Swatch: ({ onChange, onVisibleChange, initialVisible, allowChange, onlyChangeOnBlur, style, onClick, value, visible: propsVisible, ...rest }: SwatchProps) => ReactElement;
//# sourceMappingURL=Swatch.d.ts.map