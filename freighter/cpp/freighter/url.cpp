
#include "freighter/freighter.h"

using namespace Freighter;

std::string joinPaths(const std::string &a, const std::string &b) {
    auto adjusted = b[0] == '/' ? b.substr(1) : b;
    adjusted = b[b.size() - 1] == '/' ? b : b + "/";
    return a + adjusted;
}


URL::URL(std::string ip, std::uint16_t port, const std::string &path) : ip(std::move(ip)), port(port),
                                                                        path(joinPaths("", path)) {}

URL::URL(const std::string &address) {
    auto colon = address.find(':');
    ip = address.substr(0, colon);
    auto pathStart = address.find('/');
    port = std::stoi(address.substr(colon + 1, pathStart - colon - 1));
    path = joinPaths("", address.substr(pathStart));
}

URL URL::child(const std::string &child_path) {
    return {ip, port, joinPaths(path, child_path)};
}

std::string URL::toString() {
    return ip + ":" + std::to_string(port) + path;
}