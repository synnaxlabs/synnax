
export const shallowCopy = <T extends unknown>(obj: T): T => {
    if (Array.isArray(obj)) return [...obj] as T;
    if (typeof obj === "object" && obj !== null) return { ...obj } as T;
    return obj;
}