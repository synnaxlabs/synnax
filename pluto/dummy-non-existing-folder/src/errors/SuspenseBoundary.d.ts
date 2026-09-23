import { type ComponentType, type ReactElement, type ReactNode } from "react";
import { type FallbackProps } from "./Fallback";
export interface SuspenseBoundaryProps {
    loading?: ReactNode;
    FallbackComponent?: ComponentType<FallbackProps>;
    children: ReactNode;
}
export declare const SuspenseBoundary: ({ loading, FallbackComponent, children, }: SuspenseBoundaryProps) => ReactElement;
//# sourceMappingURL=SuspenseBoundary.d.ts.map