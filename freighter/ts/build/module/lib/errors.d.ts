/**
 * @description FError is an interface for an error that can be transported over
 * the network.
 */
export interface TypedError extends Error {
    discriminator: 'FreighterError';
    /**
     * @description Returns a unique type identifier for the error. Freighter uses this to
     * determine the correct decoder to use on the other end of the freighter.
     */
    type: string;
}
export declare class BaseTypedError extends Error implements TypedError {
    discriminator: 'FreighterError';
    type: string;
    constructor(message: string, type: string);
}
declare type ErrorDecoder = (encoded: string) => TypedError;
declare type ErrorEncoder = (error: TypedError) => string;
export declare const isTypedError: (error: unknown) => error is TypedError;
export declare const assertErrorType: <T>(type: string, error?: Error) => T;
export declare const UNKNOWN = "unknown";
export declare const NONE = "nil";
export declare type ErrorPayload = {
    type: string;
    data: string;
};
export declare const registerError: (props: {
    type: string;
    encode: ErrorEncoder;
    decode: ErrorDecoder;
}) => void;
export declare const encodeError: (error: unknown) => ErrorPayload;
export declare const decodeError: (payload: ErrorPayload) => TypedError | undefined;
export declare class EOF extends BaseTypedError implements TypedError {
    constructor();
}
export declare class StreamClosed extends BaseTypedError implements TypedError {
    constructor();
}
export declare class Unreachable extends BaseTypedError implements TypedError {
    constructor();
}
export {};
