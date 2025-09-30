#include "lexer.h"
#include "utils.h"
#include <string>

namespace {

bool isSpace(char c)
{
    std::string spaces = " \f\n\r\t\v";
    return spaces.find(c) != std::string::npos;
}

} // namespace 

namespace bronze
{

Token Lexer::getNextToken()
{
    char currentChar = eatNextChar();

    while (isSpace(currentChar))
    {
        currentChar = eatNextChar();
    }

    SourceLocation tokenStartLocation{source->path, this->line, this->column};
    
    for (auto &&c : singleCharTokens)
    {
        if (c == currentChar)
        {
            return Token{tokenStartLocation, static_cast<TokenKind>(c)};
        }
    }

    if (currentChar == '/')
}

} // namespace bronze
