#ifndef LEXER_H
#define LEXER_H
#include <string>
#include <vector>
#include "util.hh"


enum TokenType
{
    Number,
    Identifier,
    Equals,
    OpenParen,
    CloseParan,
    BinaryOperator,
    Unknown,
};


struct Token
{
    std::string value;
    TokenType type; 
    Token(std::string value, TokenType type)
        : value{value}, type{type}
        {}
};


std::vector<Token> tokenize(std::string& sourceCode);


#endif