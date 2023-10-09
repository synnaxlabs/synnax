#include <exception>
// Freighter.
#include "freighter/gRPC/client.h"

// Compiled protos.
#include "v1/auth.grpc.pb.h"
#include "synnax/framer/frame.h"

/// @brief The interface for the writer class.
class Writer {
public:
    /// @brief Sends one frame to the given target.
    bool write(Frame fr);

    bool commit();

    std::exception error();

    void close();

private:
};