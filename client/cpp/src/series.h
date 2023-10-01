// std.
#include <string>
#include <memory>

// Series type, able to hold generic types under the hood.
class Series
{
public:

private:
    /// @brief Holds what type of data is being used.
    std::string data_type;

    /// @brief Holds the data under the hood. 
    std::unique_ptr<void> data;
}