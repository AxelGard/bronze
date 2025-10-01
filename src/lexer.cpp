#include "lexer.h"
#include "utils.h"
#include <string>
#include <utility>

namespace
{

bool isSpace(char c)
{
    std::string spaces = " \f\n\r\t\v";
    return spaces.find(c) != std::string::npos;
}

bool isAlpha(char c)
{
    return ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z');
}

bool isNum(char c)
{
    return '0' <= c && c <= '9';
}
bool isAlnum(char c)
{
    return isAlpha(c) || isNum(c);
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
    {
        if (this->peekNextChar() != '/')
        {
            return Token{tokenStartLocation, TokenKind::Slash};
        }
        while (this->peekNextChar() != '\n' && this->peekNextChar() != '\0') // read in comment
        {
            this->eatNextChar();
        }
        return getNextToken();
    }

    if (currentChar == '=')
    {
        if (this->peekNextChar() == '=')
        {
            eatNextChar();
            return Token{tokenStartLocation, TokenKind::Equivalent};
        }
        return Token{tokenStartLocation, TokenKind::SetEqual};
    }

    if (currentChar == '&' && this->peekNextChar() == '&')
    {
        this->eatNextChar();
        return Token{tokenStartLocation, TokenKind::And};
    }

    if (currentChar == '|' && this->peekNextChar() == '|')
    {
        this->eatNextChar();
        return Token{tokenStartLocation, TokenKind::And};
    }

    if (isAlpha(currentChar)) 
    {
        std::string value{currentChar};
        while (isAlpha(this->peekNextChar()))
        {
            value += eatNextChar();
        }
        if (bronze::keywords.count(value)) // if word is keyword 
        {
            return Token{tokenStartLocation, bronze::keywords.at(value), std::move(value)};
        }
        return Token{tokenStartLocation, TokenKind::Identifier, std::move(value)};
    }

    if (isNum(currentChar)) // is int or float ?? 
    {
        std::string value{currentChar};
        while(isNum(this->peekNextChar()))
        {
            value += eatNextChar();
        }
        if (peekNextChar() != '.')
        {
            return Token{tokenStartLocation, TokenKind::KwInt, value};
        }

        value += eatNextChar(); 

        if (!isNum(this->peekNextChar())) // is float??
        {
            return Token{tokenStartLocation, TokenKind::Unknown};
        }
        while (isNum(this->peekNextChar()))
        {
            value += this->eatNextChar();
        }

        return Token{tokenStartLocation, TokenKind::KwFloat, value};
    }

    return Token{tokenStartLocation, TokenKind::Unknown};
}

} // namespace bronze
