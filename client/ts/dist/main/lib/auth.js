"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.TokenResponseSchema = exports.InsecureCredentialsSchema = exports.tokenMiddleware = void 0;
const zod_1 = require("zod");
const errors_1 = require("./errors");
const payload_1 = require("./user/payload");
const tokenMiddleware = (token) => {
    return async (md, next) => {
        try {
            md.params['Authorization'] = `Bearer ${await token()}`;
        }
        catch (err) {
            return err;
        }
        return await next(md);
    };
};
exports.tokenMiddleware = tokenMiddleware;
exports.InsecureCredentialsSchema = zod_1.z.object({
    username: zod_1.z.string(),
    password: zod_1.z.string(),
});
exports.TokenResponseSchema = zod_1.z.object({
    token: zod_1.z.string(),
    user: payload_1.UserPayloadSchema,
});
class AuthenticationClient {
    constructor(factory, creds) {
        this.client = factory.postClient();
        this.credentials = creds;
        this.authenticated = false;
        this.authenticate();
    }
    authenticate() {
        this.authenticating = new Promise((resolve, reject) => {
            this.client
                .send(AuthenticationClient.ENDPOINT, this.credentials, exports.TokenResponseSchema)
                .then(([res, err]) => {
                if (err) {
                    reject(err);
                    return;
                }
                this.token = res === null || res === void 0 ? void 0 : res.token;
                this.user = res === null || res === void 0 ? void 0 : res.user;
                this.authenticated = true;
                resolve();
            });
        });
    }
    async maybeWaitAuthenticated() {
        if (this.authenticating)
            await this.authenticating;
        this.authenticating = undefined;
    }
    middleware() {
        return (0, exports.tokenMiddleware)(async () => {
            await this.maybeWaitAuthenticated();
            if (!this.token) {
                throw new errors_1.AuthError('[auth] - attempting to authenticate without a token');
            }
            return this.token;
        });
    }
}
exports.default = AuthenticationClient;
AuthenticationClient.ENDPOINT = '/auth/login';
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiYXV0aC5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIi4uLy4uLy4uL3NyYy9saWIvYXV0aC50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiOzs7QUFLQSw2QkFBd0I7QUFFeEIscUNBQXFDO0FBQ3JDLDRDQUFnRTtBQUV6RCxNQUFNLGVBQWUsR0FBRyxDQUFDLEtBQTRCLEVBQWMsRUFBRTtJQUMxRSxPQUFPLEtBQUssRUFBRSxFQUFFLEVBQUUsSUFBSSxFQUFFLEVBQUU7UUFDeEIsSUFBSTtZQUNGLEVBQUUsQ0FBQyxNQUFNLENBQUMsZUFBZSxDQUFDLEdBQUcsVUFBVSxNQUFNLEtBQUssRUFBRSxFQUFFLENBQUM7U0FDeEQ7UUFBQyxPQUFPLEdBQUcsRUFBRTtZQUNaLE9BQU8sR0FBWSxDQUFDO1NBQ3JCO1FBQ0QsT0FBTyxNQUFNLElBQUksQ0FBQyxFQUFFLENBQUMsQ0FBQztJQUN4QixDQUFDLENBQUM7QUFDSixDQUFDLENBQUM7QUFUVyxRQUFBLGVBQWUsbUJBUzFCO0FBRVcsUUFBQSx5QkFBeUIsR0FBRyxPQUFDLENBQUMsTUFBTSxDQUFDO0lBQ2hELFFBQVEsRUFBRSxPQUFDLENBQUMsTUFBTSxFQUFFO0lBQ3BCLFFBQVEsRUFBRSxPQUFDLENBQUMsTUFBTSxFQUFFO0NBQ3JCLENBQUMsQ0FBQztBQUdVLFFBQUEsbUJBQW1CLEdBQUcsT0FBQyxDQUFDLE1BQU0sQ0FBQztJQUMxQyxLQUFLLEVBQUUsT0FBQyxDQUFDLE1BQU0sRUFBRTtJQUNqQixJQUFJLEVBQUUsMkJBQWlCO0NBQ3hCLENBQUMsQ0FBQztBQUlILE1BQXFCLG9CQUFvQjtJQVN2QyxZQUFZLE9BQTBCLEVBQUUsS0FBMEI7UUFDaEUsSUFBSSxDQUFDLE1BQU0sR0FBRyxPQUFPLENBQUMsVUFBVSxFQUFFLENBQUM7UUFDbkMsSUFBSSxDQUFDLFdBQVcsR0FBRyxLQUFLLENBQUM7UUFDekIsSUFBSSxDQUFDLGFBQWEsR0FBRyxLQUFLLENBQUM7UUFDM0IsSUFBSSxDQUFDLFlBQVksRUFBRSxDQUFDO0lBQ3RCLENBQUM7SUFFRCxZQUFZO1FBQ1YsSUFBSSxDQUFDLGNBQWMsR0FBRyxJQUFJLE9BQU8sQ0FBQyxDQUFDLE9BQU8sRUFBRSxNQUFNLEVBQUUsRUFBRTtZQUNwRCxJQUFJLENBQUMsTUFBTTtpQkFDUixJQUFJLENBQ0gsb0JBQW9CLENBQUMsUUFBUSxFQUM3QixJQUFJLENBQUMsV0FBVyxFQUNoQiwyQkFBbUIsQ0FDcEI7aUJBQ0EsSUFBSSxDQUFDLENBQUMsQ0FBQyxHQUFHLEVBQUUsR0FBRyxDQUFDLEVBQUUsRUFBRTtnQkFDbkIsSUFBSSxHQUFHLEVBQUU7b0JBQ1AsTUFBTSxDQUFDLEdBQUcsQ0FBQyxDQUFDO29CQUNaLE9BQU87aUJBQ1I7Z0JBQ0QsSUFBSSxDQUFDLEtBQUssR0FBRyxHQUFHLGFBQUgsR0FBRyx1QkFBSCxHQUFHLENBQUUsS0FBSyxDQUFDO2dCQUN4QixJQUFJLENBQUMsSUFBSSxHQUFHLEdBQUcsYUFBSCxHQUFHLHVCQUFILEdBQUcsQ0FBRSxJQUFJLENBQUM7Z0JBQ3RCLElBQUksQ0FBQyxhQUFhLEdBQUcsSUFBSSxDQUFDO2dCQUMxQixPQUFPLEVBQUUsQ0FBQztZQUNaLENBQUMsQ0FBQyxDQUFDO1FBQ1AsQ0FBQyxDQUFDLENBQUM7SUFDTCxDQUFDO0lBRU8sS0FBSyxDQUFDLHNCQUFzQjtRQUNsQyxJQUFJLElBQUksQ0FBQyxjQUFjO1lBQUUsTUFBTSxJQUFJLENBQUMsY0FBYyxDQUFDO1FBQ25ELElBQUksQ0FBQyxjQUFjLEdBQUcsU0FBUyxDQUFDO0lBQ2xDLENBQUM7SUFFRCxVQUFVO1FBQ1IsT0FBTyxJQUFBLHVCQUFlLEVBQUMsS0FBSyxJQUFJLEVBQUU7WUFDaEMsTUFBTSxJQUFJLENBQUMsc0JBQXNCLEVBQUUsQ0FBQztZQUNwQyxJQUFJLENBQUMsSUFBSSxDQUFDLEtBQUssRUFBRTtnQkFDZixNQUFNLElBQUksa0JBQVMsQ0FDakIscURBQXFELENBQ3RELENBQUM7YUFDSDtZQUNELE9BQU8sSUFBSSxDQUFDLEtBQUssQ0FBQztRQUNwQixDQUFDLENBQUMsQ0FBQztJQUNMLENBQUM7O0FBcERILHVDQXFEQztBQXBEZ0IsNkJBQVEsR0FBRyxhQUFhLENBQUMifQ==