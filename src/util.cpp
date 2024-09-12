#include "util.hh"

std::vector<std::string> split(std::string s, char delimiter) {
    std::stringstream ss{s};
    std::vector<std::string> tokens;
    std::string token;
    while (getline(ss, token, delimiter))
    {
        if (token == " ") continue;
        tokens.push_back(token);
    } 
    return tokens;
}


bool isNumber(const std::string& s)
{
    std::string::const_iterator it = s.begin();
    while (it != s.end() && std::isdigit(*it)) ++it;
    return !s.empty() && it == s.end();
}