from jwt import de
from freighter import UnaryClient, Payload


class InsecureCredentials(Payload):
    username: str
    password: str


class AuthenticationClient:
    _ENDPOINT = "/auth/login"

    transport: UnaryClient
    username: str
    password: str
    token: str

    def __init__(self,
                 transport: UnaryClient,
                 username: str,
                 password: str,
                 ) -> None:
        self.transport = transport
        self.username = username
        self.password = password

    def authenticate(self) -> None:
        self.transport.send(self._ENDPOINT,
                            InsecureCredentials(self.username, self.password))

