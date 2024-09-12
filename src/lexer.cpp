#include "lexer.hh"


std::vector<Token> tokenize(std::string& sourceCode)
{
    std::vector<Token> tokens;
    std::vector<std::string> src = split(sourceCode);
    while (! src.empty())
    {
        std::string t = src.front();
        if ( t == "(")
        {
            tokens.push_back(Token(t, TokenType::OpenParen));
        }
        else if (t == ")")
        {
            tokens.push_back(Token(t, TokenType::CloseParan));
        }
        else if (t == ")")
        {
            tokens.push_back(Token(t, TokenType::CloseParan));
        }
        else if (t == "=")
        {
            tokens.push_back(Token(t, TokenType::Equals));
        }
        else if (t == "+")
        {
            tokens.push_back(Token(t, TokenType::BinaryOperator));
        }
        else if (isNumber(t)) 
        {
            tokens.push_back(Token(t, TokenType::Number));
        }
        else
        {
            tokens.push_back(Token(t, TokenType::Unknown));
        }
        

        src.erase(src.begin());
    }

    return tokens;
}
