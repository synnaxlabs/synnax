import { BaseTypedError, registerError } from '@synnaxlabs/freighter';
import { z } from 'zod';
const _FREIGHTER_EXCEPTION_TYPE = 'synnax.api.errors';
const APIErrorPayloadSchema = z.object({
    type: z.string(),
    error: z.record(z.unknown()),
});
var APIErrorType;
(function (APIErrorType) {
    APIErrorType["General"] = "general";
    APIErrorType["Nil"] = "nil";
    APIErrorType["Parse"] = "parse";
    APIErrorType["Auth"] = "auth";
    APIErrorType["Unexpected"] = "unexpected";
    APIErrorType["Validation"] = "validation";
    APIErrorType["Query"] = "query";
    APIErrorType["Route"] = "route";
})(APIErrorType || (APIErrorType = {}));
class BaseError extends BaseTypedError {
    constructor(message) {
        super(message, _FREIGHTER_EXCEPTION_TYPE);
    }
}
/**
 * Raised when a validation error occurs.
 */
export class ValidationError extends BaseError {
    fields;
    constructor(fieldsOrMessage) {
        if (typeof fieldsOrMessage === 'string') {
            super(fieldsOrMessage);
            this.fields = [];
        }
        else if (Array.isArray(fieldsOrMessage)) {
            super(fieldsOrMessage
                .map((field) => `${field.field}: ${field.message}`)
                .join('\n'));
            this.fields = fieldsOrMessage;
        }
        else {
            super(`${fieldsOrMessage.field}: ${fieldsOrMessage.message}`);
            this.fields = [fieldsOrMessage];
        }
    }
}
/**
 * GeneralError is raised when a general error occurs.
 */
export class GeneralError extends BaseError {
}
/**
 * ParseError is raised when a parse error occurs.
 */
export class ParseError extends BaseError {
}
/**
 * AuthError is raised when an authentication error occurs.
 */
export class AuthError extends BaseError {
}
/**
 * UnexpectedError is raised when an unexpected error occurs.
 */
export class UnexpectedError extends BaseError {
}
/**
 * QueryError is raised when a query error occurs.
 */
export class QueryError extends BaseError {
}
/**
 * RouteError is raised when a routing error occurs.
 */
export class RouteError extends BaseError {
    path;
    constructor(message, path) {
        super(message);
        this.path = path;
    }
}
/**
 * Raised when time-series data is not contiguous.
 */
export class ContiguityError extends BaseError {
}
const parsePayload = (payload) => {
    switch (payload.type) {
        case APIErrorType.General:
            return new GeneralError(payload.error.message);
        case APIErrorType.Parse:
            return new ParseError(payload.error.message);
        case APIErrorType.Auth:
            return new AuthError(payload.error.message);
        case APIErrorType.Unexpected:
            return new UnexpectedError(JSON.stringify(payload.error));
        case APIErrorType.Validation:
            return new ValidationError(payload.error.fields);
        case APIErrorType.Query:
            return new QueryError(payload.error.message);
        case APIErrorType.Route:
            return new RouteError(payload.error.path, payload.error.message);
        default:
            return undefined;
    }
};
const decode = (encoded) => {
    return parsePayload(APIErrorPayloadSchema.parse(JSON.parse(encoded)));
};
const encode = () => {
    throw new Error('Not implemented');
};
registerError({ type: _FREIGHTER_EXCEPTION_TYPE, encode, decode });
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZXJyb3JzLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiLi4vLi4vLi4vc3JjL2xpYi9lcnJvcnMudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsT0FBTyxFQUFFLGNBQWMsRUFBRSxhQUFhLEVBQUUsTUFBTSx1QkFBdUIsQ0FBQztBQUN0RSxPQUFPLEVBQUUsQ0FBQyxFQUFFLE1BQU0sS0FBSyxDQUFDO0FBRXhCLE1BQU0seUJBQXlCLEdBQUcsbUJBQW1CLENBQUM7QUFFdEQsTUFBTSxxQkFBcUIsR0FBRyxDQUFDLENBQUMsTUFBTSxDQUFDO0lBQ3JDLElBQUksRUFBRSxDQUFDLENBQUMsTUFBTSxFQUFFO0lBQ2hCLEtBQUssRUFBRSxDQUFDLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQyxPQUFPLEVBQUUsQ0FBQztDQUM3QixDQUFDLENBQUM7QUFJSCxJQUFLLFlBU0o7QUFURCxXQUFLLFlBQVk7SUFDZixtQ0FBbUIsQ0FBQTtJQUNuQiwyQkFBVyxDQUFBO0lBQ1gsK0JBQWUsQ0FBQTtJQUNmLDZCQUFhLENBQUE7SUFDYix5Q0FBeUIsQ0FBQTtJQUN6Qix5Q0FBeUIsQ0FBQTtJQUN6QiwrQkFBZSxDQUFBO0lBQ2YsK0JBQWUsQ0FBQTtBQUNqQixDQUFDLEVBVEksWUFBWSxLQUFaLFlBQVksUUFTaEI7QUFPRCxNQUFNLFNBQVUsU0FBUSxjQUFjO0lBQ3BDLFlBQVksT0FBZTtRQUN6QixLQUFLLENBQUMsT0FBTyxFQUFFLHlCQUF5QixDQUFDLENBQUM7SUFDNUMsQ0FBQztDQUNGO0FBRUQ7O0dBRUc7QUFDSCxNQUFNLE9BQU8sZUFBZ0IsU0FBUSxTQUFTO0lBQzVDLE1BQU0sQ0FBVTtJQUVoQixZQUFZLGVBQXlDO1FBQ25ELElBQUksT0FBTyxlQUFlLEtBQUssUUFBUSxFQUFFO1lBQ3ZDLEtBQUssQ0FBQyxlQUFlLENBQUMsQ0FBQztZQUN2QixJQUFJLENBQUMsTUFBTSxHQUFHLEVBQUUsQ0FBQztTQUNsQjthQUFNLElBQUksS0FBSyxDQUFD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