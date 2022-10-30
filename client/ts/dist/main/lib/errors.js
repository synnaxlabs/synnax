"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.ContiguityError = exports.RouteError = exports.QueryError = exports.UnexpectedError = exports.AuthError = exports.ParseError = exports.GeneralError = exports.ValidationError = void 0;
const freighter_1 = require("@synnaxlabs/freighter");
const zod_1 = require("zod");
const _FREIGHTER_EXCEPTION_TYPE = 'synnax.api.errors';
const APIErrorPayloadSchema = zod_1.z.object({
    type: zod_1.z.string(),
    error: zod_1.z.record(zod_1.z.unknown()),
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
class BaseError extends freighter_1.BaseTypedError {
    constructor(message) {
        super(message, _FREIGHTER_EXCEPTION_TYPE);
    }
}
/**
 * Raised when a validation error occurs.
 */
class ValidationError extends BaseError {
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
exports.ValidationError = ValidationError;
/**
 * GeneralError is raised when a general error occurs.
 */
class GeneralError extends BaseError {
}
exports.GeneralError = GeneralError;
/**
 * ParseError is raised when a parse error occurs.
 */
class ParseError extends BaseError {
}
exports.ParseError = ParseError;
/**
 * AuthError is raised when an authentication error occurs.
 */
class AuthError extends BaseError {
}
exports.AuthError = AuthError;
/**
 * UnexpectedError is raised when an unexpected error occurs.
 */
class UnexpectedError extends BaseError {
}
exports.UnexpectedError = UnexpectedError;
/**
 * QueryError is raised when a query error occurs.
 */
class QueryError extends BaseError {
}
exports.QueryError = QueryError;
/**
 * RouteError is raised when a routing error occurs.
 */
class RouteError extends BaseError {
    constructor(message, path) {
        super(message);
        this.path = path;
    }
}
exports.RouteError = RouteError;
/**
 * Raised when time-series data is not contiguous.
 */
class ContiguityError extends BaseError {
}
exports.ContiguityError = ContiguityError;
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
(0, freighter_1.registerError)({ type: _FREIGHTER_EXCEPTION_TYPE, encode, decode });
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZXJyb3JzLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiLi4vLi4vLi4vc3JjL2xpYi9lcnJvcnMudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6Ijs7O0FBQUEscURBQXNFO0FBQ3RFLDZCQUF3QjtBQUV4QixNQUFNLHlCQUF5QixHQUFHLG1CQUFtQixDQUFDO0FBRXRELE1BQU0scUJBQXFCLEdBQUcsT0FBQyxDQUFDLE1BQU0sQ0FBQztJQUNyQyxJQUFJLEVBQUUsT0FBQyxDQUFDLE1BQU0sRUFBRTtJQUNoQixLQUFLLEVBQUUsT0FBQyxDQUFDLE1BQU0sQ0FBQyxPQUFDLENBQUMsT0FBTyxFQUFFLENBQUM7Q0FDN0IsQ0FBQyxDQUFDO0FBSUgsSUFBSyxZQVNKO0FBVEQsV0FBSyxZQUFZO0lBQ2YsbUNBQW1CLENBQUE7SUFDbkIsMkJBQVcsQ0FBQTtJQUNYLCtCQUFlLENBQUE7SUFDZiw2QkFBYSxDQUFBO0lBQ2IseUNBQXlCLENBQUE7SUFDekIseUNBQXlCLENBQUE7SUFDekIsK0JBQWUsQ0FBQTtJQUNmLCtCQUFlLENBQUE7QUFDakIsQ0FBQyxFQVRJLFlBQVksS0FBWixZQUFZLFFBU2hCO0FBT0QsTUFBTSxTQUFVLFNBQVEsMEJBQWM7SUFDcEMsWUFBWSxPQUFlO1FBQ3pCLEtBQUssQ0FBQyxPQUFPLEVBQUUseUJBQXlCLENBQUMsQ0FBQztJQUM1QyxDQUFDO0NBQ0Y7QUFFRDs7R0FFRztBQUNILE1BQWEsZUFBZ0IsU0FBUSxTQUFTO0lBRzVDLFlBQVksZUFBeUM7UUFDbkQsSUFBSSxPQUFPLGVBQWUsS0FBSyxRQUFRLEVBQUU7WUFDdkMsS0FBSyxDQUFDLGVBQWUsQ0FBQyxDQUFDO1lBQ3ZCLElBQUksQ0FBQyxNQUFNLEdBQUcsRUFBRSxDQUFDO1NBQ2xCO2FBQU0sSUFBSSxLQUFLLENBQUMsT0FBTyxDQUFDLGVBQWUsQ0FBQyxFQUFFO1lBQ3pDLEtBQUssQ0FDSCxlQUFlO2lCQUNaLEdBQUcsQ0FBQyxDQUFDLEtBQUssRUFBRSxFQUFFLENB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