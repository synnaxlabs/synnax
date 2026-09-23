import { DISPLAY } from "./constants";
interface DisplaySelectorProps {
    display: (typeof DISPLAY)[number][];
    setDisplay: (display: (typeof DISPLAY)[number][]) => void;
}
export declare const DisplaySelector: ({ display, setDisplay }: DisplaySelectorProps) => import("react").JSX.Element;
export {};
//# sourceMappingURL=DisplaySelector.d.ts.map