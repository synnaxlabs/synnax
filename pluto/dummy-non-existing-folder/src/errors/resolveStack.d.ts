export interface ResolvedStack {
    stack: string;
    componentStack: string | null;
}
/**
 * Resolves the source-map-mapped form of an Error's stack and (optionally) its React
 * component stack into clean, human-readable text. Each leg is resolved independently:
 * if a leg cannot be resolved (e.g. source maps unavailable, parser cannot recognize
 * the stack format), that leg degrades to its raw input and a warning is surfaced via
 * console.warn, while the other leg's resolved output is preserved. Silently swallowing
 * would mask real deployment problems (missing maps, broken middleware, etc.), so
 * failures are always logged.
 */
export declare const resolveStack: (error: Error, componentStack: string | null) => Promise<ResolvedStack>;
//# sourceMappingURL=resolveStack.d.ts.map