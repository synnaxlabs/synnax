import { type FC, type ReactNode } from "react";
export type CanDisabledProps<T extends object & {
    children?: ReactNode;
}> = T & {
    disabled?: boolean;
};
export declare const canDisable: <T extends object & {
    children?: ReactNode;
}>(C: FC<T>) => FC<CanDisabledProps<T>>;
//# sourceMappingURL=canDisable.d.ts.map