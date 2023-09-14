#pragma once

/// Local headers.
#include "types.h"

/// @brief Interface for client.
class Client
{
public:
    /// @brief Interface for unary send.
    virtual Response send(Target target, Request& request) = 0;

    /// @brief Interface for stream.
    virtual Stream stream(Target target) = 0;
};