import {
  LoginRequest,
  AuthenticationResponse,
  AuthenticationTransport,
  Token,
} from "./auth";
import { HTTPClient } from "../http/client";

class HTTPAuthenticationTransport implements AuthenticationTransport {
  private client: HTTPClient;

  async login(req: LoginRequest): Promise<AuthenticationResponse> {
    return await this.client.Post<LoginRequest, AuthenticationResponse>({
      endpoint: "/auth/login",
      req: req,
    });
  }

  bindCredentials(token: Token) {
    this.client.token = { headers: { Authorization: `Bearer ${token}` } };
  }
}
