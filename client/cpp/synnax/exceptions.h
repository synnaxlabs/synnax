#pragma once

class QueryError : public std::exception {
public:
    QueryError(std::string message) : message(message) {}

    std::string message;

    char *what() {
        return message.c_str();
    }
};


