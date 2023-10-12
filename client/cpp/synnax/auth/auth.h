#pragma once

namespace Auth {
    class Auth {
    public:
        std::string username;
        std::string password;

        Auth(std::string username, std::string password) :
                username(std::move(username)), password(std::move(password)) {}
    };
}