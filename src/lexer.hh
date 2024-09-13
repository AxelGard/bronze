#ifndef LEXER_H
#define LEXER_H
#include <string>
#include <vector>
#include "util.hh"
#include <map>


enum TokenType
{
    Number,
    Identifier,
    Equals,
    OpenParen,
    CloseParan,
    BinaryOperator,
    Unknown,
    Keywords,
};


enum Keyword
{
    Func,
    If,
    Else,
    For,
    Import,

};

std::map<std::string, Keyword> keywordsMap = {
    {"func",Keyword::Func},
    {"if",Keyword::If},
    {"else",Keyword::Else},
    {"for",Keyword::For},
    {"import",Keyword::Import},
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
bool isKeyword(std::string word);

#endif
