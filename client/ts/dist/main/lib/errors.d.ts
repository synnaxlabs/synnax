import { BaseTypedError } from '@synnaxlabs/freighter';
export declare type Field = {
    field: string;
    message: string;
};
declare class BaseError extends BaseTypedError {
    constructor(message: string);
}
/**
 * Raised when a validation error occurs.
 */
export declare class ValidationError extends BaseError {
    fields: Field[];
    constructor(fieldsOrMessage: string | Field[] | Field);
}
/**
 * GeneralError is raised when a general error occurs.
 */
export declare class GeneralError extends BaseError {
}
/**
 * ParseError is raised when a parse error occurs.
 */
export declare class ParseError extends BaseError {
}
/**
 * AuthError is raised when an authentication error occurs.
 */
export declare class AuthError extends BaseError {
}
/**
 * UnexpectedError is raised when an unexpected error occurs.
 */
export declare class UnexpectedError extends BaseError {
}
/**
 * QueryError is raised when a query error occurs.
 */
export declare class QueryError extends BaseError {
}
/**
 * RouteError is raised when a routing error occurs.
 */
export declare class RouteError extends BaseError {
    path: string;
    constructor(message: string, path: string);
}
/**
 * Raised when time-series data is not contiguous.
 */
export declare class ContiguityError extends BaseError {
}
export {};
