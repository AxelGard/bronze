#include <iostream>
#include <string>

namespace bz
{
    void print(std::string text)
    {
        std::cout << text << std::endl;
    }

    enum Error 
    {
        Not = 0,
        Range_Error = 1,
    };

}


int main()
{
    if (int(bz::Error::Not))
    {
        bz::print("hello");
    }
    return 0;
}
