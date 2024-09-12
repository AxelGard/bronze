#ifndef LEXER_H
#define LEXER_H
#include <string>
#include <vector>


enum TokenType
{
    Number,
    Identifier,
    Equals,
    OpenParen,
    CloseParan,
    BinaryOperator,
    Let,
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


std::vector<std::string> split(std::string str, char on = ' ');
std::vector<Token> tokenize(std::string& sourceCode);


#endif