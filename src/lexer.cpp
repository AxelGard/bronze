#include "lexer.hh"


bool isKeyword(std::string word)
{
    for (auto const& w : keywordsMap)
    {
        if (w.first == word) return true;
    }
    return false;
}

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
        else if (isKeyword(t))
        {
            tokens.push_back(Token(t, TokenType::Keywords));
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
